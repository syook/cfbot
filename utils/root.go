package utils

import (
	"bytes"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	csr "github.com/cloudflare/cfssl/csr"
	cloudflare "github.com/cloudflare/cloudflare-go"
	"github.com/fatih/color"
	"github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
	"github.com/syook/cfbot/structs"
)

var (
	// successFlag success flag
	successFlag = color.GreenString("✔ ")
	// errorFlag error flag
	errorFlag = color.RedString("✗ ")
	//warningFlag warning flag
	warningFlag = color.YellowString("� ")
)

var destinationValue string
var configValues structs.Configs
var initialRun bool

//TODO: set this buffer as argument from the flags?
const bufferHours float64 = 48

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
	if destinationValue == "" {
		homePath, err := homedir.Dir()
		directoryPath := filepath.Join(homePath, "cfbot")
		Check(err)
		return directoryPath
	}
	directoryPath = destinationValue
	return directoryPath
}

func saveConfigsJSON(configValues structs.Configs) {
	directoryPath := getDestinationPath()
	jsonValue, err := json.Marshal(configValues)
	Check(err)
	configFile := filepath.Join(directoryPath, "cfbot.json")
	configJSON, err := os.OpenFile(configFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0664)
	Check(err)
	_, err = configJSON.Write(jsonValue)
	Check(err)
}

//generateCertificate helps you to get all the certificates from cloudflare
func generateCertificate(saveConfigs bool) {
	directoryPath := getDestinationPath()

	//TODO: this type rsa/ecdsa and also key length needs to be added as configs via flags
	privateKeyRequest := csr.KeyRequest{A: "rsa", S: 2048}
	newCertificateRequest := csr.CertificateRequest{CN: "Cloudflare", Hosts: configValues.Hostnames, KeyRequest: &privateKeyRequest}

	csrValue, key, err := csr.ParseRequest(&newCertificateRequest)
	Check(err)

	var certOutBuffer bytes.Buffer

	keyFile := filepath.Join(directoryPath, "key.pem")
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

	certificateFile := filepath.Join(directoryPath, "certificate.pem")
	certOut, err := os.OpenFile(certificateFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	Check(err)

	//write the certificate we got to the file
	certOut.WriteString(responseCertificate.Certificate)

	if saveConfigs {
		saveConfigsJSON(configValues)
	}
}

func checkValidityOfCertificate() bool {
	directoryPath := getDestinationPath()
	certificateFile := filepath.Join(directoryPath, "certificate.pem")
	certificateValue, err := ioutil.ReadFile(certificateFile)
	Check(err)
	pemBlock, _ := pem.Decode(certificateValue)
	if pemBlock == nil {
		Check(errors.New("failed to parse certificate PEM"))
	}
	cert, err := x509.ParseCertificate(pemBlock.Bytes)
	Check(err)
	currentTime := time.Now()
	validityLeft := cert.NotAfter.Sub(currentTime).Hours()
	if validityLeft < bufferHours {
		//if the validity is less than the buffer hours, return true so that new certificates are generated
		return true
	}
	return false
}

//verifyDirectoryExists checks if certs directory exists, if not it creates the directory in the home directory or the given destination path
func verifyDirectoryExists() {
	directoryPath := getDestinationPath()
	_, err := os.Stat(directoryPath)
	if os.IsNotExist(err) {
		//if folder does not exist create a new folder
		err := os.MkdirAll(directoryPath, 0755)
		Check(err)
	}
}

//checkInitialRunAndCreateCertificate is used to complete the whole process of verifying existing directory and getting a new certificate
func checkInitialRunAndCreateCertificate() {
	verifyDirectoryExists()
	if initialRun {
		//this means you're running it for the first time, save the config values
		generateCertificate(true)
		return
	}
	//this usually means you're supposed to get new certificates since the old ones are about to expire
	//TODO:check if certificates need to be renewed && backup existing certificates && then try to get new certificates
	certificatesNeedsRenewal := checkValidityOfCertificate()
	if certificatesNeedsRenewal {
		generateCertificate(false)
	} else {
		fmt.Printf("%s Certificates are still valid within the buffer time, not getting new certificates", warningFlag)
	}
}

//validateFlags is used to validate all the required flags are passed in
func validateFlags() {
	//Checking if mandatory flags are set or not since along with viper we don't have a workaround
	if configValues.AuthServiceKey == "" {
		Check(errors.New("flag --auth not set"))
	}
	if len(configValues.Hostnames) == 0 {
		Check(errors.New("flag --hostnames not set"))
	}
	checkInitialRunAndCreateCertificate()
}

//Cfbot function is the entrypoint to the application
func Cfbot() {
	destination := viper.GetString("destination")
	authServiceKey := viper.GetString("auth")
	hosts := viper.GetStringSlice("hostnames")
	validity := viper.GetInt("validity")
	initialRun = viper.GetBool("init")
	// fmt.Println(authServiceKey, hosts, validity)
	destinationValue = destination
	configValues.AuthServiceKey = authServiceKey
	configValues.Hostnames = hosts
	configValues.Validity = validity
	validateFlags()
}
