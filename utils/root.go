package utils

import (
	"bytes"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	csr "github.com/cloudflare/cfssl/csr"
	cloudflare "github.com/cloudflare/cloudflare-go"
	"github.com/mitchellh/go-homedir"
	"github.com/syook/cfbot/structs"
)

var destination string

func er(msg interface{}) {
	fmt.Println("Error:", msg)
	os.Exit(1)
}

//Check is used as a centralized place to handle errors
func Check(e error) {
	if e != nil {
		// panic(e)
		er(e)
	}
}

func getDestinationPath() string {
	var directoryPath string
	if destination != "" {
		directoryPath, err := homedir.Dir()
		Check(err)
		return directoryPath
	}
	directoryPath = destination
	return directoryPath
}

func saveConfigsJSON(configValues structs.Configs) {
	directoryPath := getDestinationPath()
	jsonValue, err := json.Marshal(configValues)
	Check(err)
	configFile := filepath.Join(directoryPath, "cfbot.json")
	configJSON, err := os.OpenFile(configFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0700)
	Check(err)
	configJSON.Write(jsonValue)
}

//generateCertificate helps you to get all the certificates from cloudflare
func generateCertificate(configValues structs.Configs, saveConfigs bool) {

	directoryPath := getDestinationPath()
	folderPath := filepath.Join(directoryPath, "certs")

	//TODO: this type rsa/ecdsa and also key length needs to be added as configs via flags
	privateKeyRequest := csr.KeyRequest{A: "rsa", S: 2048}
	newCertificateRequest := csr.CertificateRequest{CN: "Cloudflare", Hosts: configValues.Hostnames, KeyRequest: &privateKeyRequest}

	csrValue, key, err := csr.ParseRequest(&newCertificateRequest)
	Check(err)

	var certOutBuffer bytes.Buffer

	keyFile := filepath.Join(folderPath, "key.pem")
	keyOut, err := os.OpenFile(keyFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	Check(err)

	certVal, _ := pem.Decode(csrValue)
	keyVal, _ := pem.Decode(key)

	if err := pem.Encode(&certOutBuffer, &pem.Block{Type: certVal.Type, Bytes: certVal.Bytes}); err != nil {
		fmt.Println("Failed to decode data to certVal")
	}

	if err := pem.Encode(keyOut, &pem.Block{Type: keyVal.Type, Bytes: keyVal.Bytes}); err != nil {
		fmt.Println("Failed to write data to key.pem")
	}

	newCertificate := cloudflare.OriginCACertificate{Hostnames: configValues.Hostnames, RequestType: "origin-rsa", RequestValidity: configValues.Validity, CSR: certOutBuffer.String()}

	api, err := cloudflare.NewWithUserServiceKey(configValues.AuthServiceKey)

	Check(err)
	responseCertificate, err := api.CreateOriginCertificate(newCertificate)
	Check(err)

	certificateFile := filepath.Join(folderPath, "certificate.pem")
	certOut, err := os.OpenFile(certificateFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	Check(err)

	//write the certificate we got to the file
	certOut.WriteString(responseCertificate.Certificate)

	if saveConfigs {
		saveConfigsJSON(configValues)
	}
}

//verifyDirectoryExists checks if certs directory exists, if not it creates the directory in the home directory
func verifyDirectoryExists() bool {

	directoryPath := getDestinationPath()
	folderPath := filepath.Join(directoryPath, "certs")
	_, err := os.Stat(folderPath)

	if os.IsNotExist(err) {
		//if folder does not exist create a new folder
		err := os.MkdirAll(folderPath, 0755)
		Check(err)
		//this means it did not exist we created it
		return false
	}

	//this means it already existed
	return true
}

func checkValidityOfCertificate() {

}

//CheckValuesAndCreateCertificate is used to complete the whole process of verifying existing directory and getting a new certificate
func CheckValuesAndCreateCertificate(configValues structs.Configs) {
	directoryExisted := verifyDirectoryExists()
	if directoryExisted {
		//this usually means you're about to get new certificates
		//TODO:check if certificates need to be renewed && backup existing certificates && then try to get new certificates
		generateCertificate(configValues, false)
	} else {
		//this means you're running it for the first time, save the config values
		generateCertificate(configValues, true)
	}
}

//ValidateFlags is used to validate all the required flags are passed in
func ValidateFlags(configValues structs.Configs, destinationValue string) {
	//Checking if mandatory flags are set or not since along with viper we don't have a workaround
	if configValues.AuthServiceKey == "" {
		Check(errors.New("flag --auth not set"))
	}
	if len(configValues.Hostnames) == 0 {
		Check(errors.New("flag --hostnames not set"))
	}
	destination = destinationValue
	CheckValuesAndCreateCertificate(configValues)
}
