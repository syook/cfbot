package utils

import (
	"bytes"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"

	csr "github.com/cloudflare/cfssl/csr"
	cloudflare "github.com/cloudflare/cloudflare-go"
	"github.com/mitchellh/go-homedir"
)

func check(e error) {
	if e != nil {
		panic(e)
	}
}

//generateCertificate helps you to get all the certificates from cloudflare
func generateCertificate(authUserService string, hostnames []string, validity int, saveConfigs bool) {
	home, err := homedir.Dir()
	check(err)

	//TODO:
	//By default now adding certs and creating a directory for cfbot at the home directory level, need to make this dynamic
	folderPath := filepath.Join(home, "certs")

	privateKeyRequest := csr.KeyRequest{A: "rsa", S: 2048}
	newCertificateRequest := csr.CertificateRequest{CN: "Cloudflare", Hosts: hostnames, KeyRequest: &privateKeyRequest}

	csrValue, key, err := csr.ParseRequest(&newCertificateRequest)
	check(err)

	var certOutBuffer bytes.Buffer

	keyFile := filepath.Join(folderPath, "key.pem")
	keyOut, err := os.OpenFile(keyFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	check(err)

	certVal, _ := pem.Decode(csrValue)
	keyVal, _ := pem.Decode(key)

	if err := pem.Encode(&certOutBuffer, &pem.Block{Type: certVal.Type, Bytes: certVal.Bytes}); err != nil {
		fmt.Println("Failed to decode data to certVal")
	}

	if err := pem.Encode(keyOut, &pem.Block{Type: keyVal.Type, Bytes: keyVal.Bytes}); err != nil {
		fmt.Println("Failed to write data to key.pem")
	}

	newCertificate := cloudflare.OriginCACertificate{Hostnames: hostnames, RequestType: "origin-rsa", RequestValidity: validity, CSR: certOutBuffer.String()}

	api, err := cloudflare.NewWithUserServiceKey(authUserService)

	check(err)
	responseCertificate, err := api.CreateOriginCertificate(newCertificate)
	check(err)

	certificateFile := filepath.Join(folderPath, "certificate.pem")
	certOut, err := os.OpenFile(certificateFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	certOut.WriteString(responseCertificate.Certificate)
	check(err)

	if saveConfigs {
		//TODO: save the configs to ~/certs/cfbot.json
	}
}

//verifyDirectoryExists checks if certs directory exists, if not it creates the directory in the home directory
func verifyDirectoryExists() bool {
	home, err := homedir.Dir()
	check(err)

	folderPath := filepath.Join(home, "certs")
	//TODO:
	//By default now adding certs and creating a directory for cfbot at the home directory level, need to make this dynamic
	_, err = os.Stat(folderPath)

	if os.IsNotExist(err) {
		//if folder does not exist create a new folder
		err := os.MkdirAll(folderPath, 0755)
		check(err)
		//this means it did not exist we created it
		return false
	}
	// fmt.Println(folderInfo)
	//this means it already existed
	return true
}

//CheckValuesAndCreateCertificate is used to complete the whole process of verifying existing directory and getting a new certificate
func CheckValuesAndCreateCertificate(authUserService string, hostnames []string, validity int) {
	directoryExisted := verifyDirectoryExists()
	if directoryExisted {
		//this usually means you're getting new certificates
		generateCertificate(authUserService, hostnames, validity, false)
	} else {
		//this means you're running it for the first time, save the config values
		generateCertificate(authUserService, hostnames, validity, true)
	}
}
