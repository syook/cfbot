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
	"os/exec"
	"path/filepath"
	"time"

	csr "github.com/cloudflare/cfssl/csr"
	cloudflare "github.com/cloudflare/cloudflare-go"
	"github.com/fatih/color"
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

const cfbotFolderPath string = "/etc/cfbot"
const cfbotLiveFolderPath string = "/etc/cfbot/live"

// var destinationValue string
var configValues structs.Configs
var initialRun bool

//TODO: set this buffer as argument from the flags?
const bufferHours float64 = 48

func er(msg interface{}) {
	fmt.Printf("%s Error: %s", errorFlag, msg)
	os.Exit(1)
}

//Check is used as a centralized place to handle errors
func Check(e error) {
	if e != nil {
		// panic(e)
		er(e)
	}
}

//executePostRenewCommnd is used to execute the post renew command given by the user
func executePostRenewCommnd() {
	fmt.Printf("%s Executing post renew command\n", successFlag)
	fmt.Printf(configValues.PostRenewCommand)
	cmd := exec.Command("bash", "-c", configValues.PostRenewCommand)

	//ignore the output from the command
	_, err := cmd.Output()
	Check(err)

	fmt.Printf("%s executed post renew command\n", successFlag)
}

//revokePreviousCertificate is used to revoke the certificate that was replaced right now from cloudflare
func revokePreviousCertificate() {
	api, err := cloudflare.NewWithUserServiceKey(configValues.AuthServiceKey)
	Check(err)

	_, err = api.RevokeOriginCertificate(configValues.PreviousCertificateID)
	Check(err)

	fmt.Printf("%s Revoked Old Certificates from cloudflare\n", successFlag)
}

//addCron is used to add the cron job on the initial run
func addCron() {
	crontabDirectory := "/etc/cron.d"
	cronFilePath := filepath.Join(crontabDirectory, "cfbot")
	//open file with rw-r--r--
	cronFile, err := os.OpenFile(cronFilePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	Check(err)
	cronFileString := fmt.Sprintf(`
# /etc/cron.d/cfbot: crontab entries for the cfbot package
#
# Upstream recommends attempting renewal twice a day
#
# Renewal will only occur if expiration is within %f hours.
SHELL=/bin/bash
PATH=/usr/local/sbin:/usr/local/bin:/sbin:/bin:/usr/sbin:/usr/bin

0 */12 * * * root test -x /usr/bin/cfbot && cfbot >> /etc/cfbot/debug.log 2>&1
`, bufferHours)
	cronFile.WriteString(cronFileString)
}

func saveConfigsJSON() {
	jsonValue, err := json.Marshal(configValues)
	Check(err)
	configFile := filepath.Join(cfbotFolderPath, "cfbot.json")
	//open file with rw-rw-r--
	configJSON, err := os.OpenFile(configFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0664)
	Check(err)
	_, err = configJSON.Write(jsonValue)
	Check(err)
}

//generateCertificate helps you to get all the certificates from cloudflare
func generateCertificate() {
	//TODO: this type rsa/ecdsa and also key length needs to be added as configs via flags
	privateKeyRequest := csr.KeyRequest{A: "rsa", S: 2048}
	newCertificateRequest := csr.CertificateRequest{CN: "Cloudflare", Hosts: configValues.Hostnames, KeyRequest: &privateKeyRequest}

	csrValue, key, err := csr.ParseRequest(&newCertificateRequest)
	Check(err)

	var certOutBuffer bytes.Buffer

	keyFile := filepath.Join(cfbotLiveFolderPath, "key.pem")
	keyOut, err := os.OpenFile(keyFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	Check(err)

	certVal, _ := pem.Decode(csrValue)
	keyVal, _ := pem.Decode(key)

	if err := pem.Encode(&certOutBuffer, &pem.Block{Type: certVal.Type, Bytes: certVal.Bytes}); err != nil {
		Check(errors.New("Failed to decode data to certVal"))
	}

	if err := pem.Encode(keyOut, &pem.Block{Type: keyVal.Type, Bytes: keyVal.Bytes}); err != nil {
		Check(errors.New("Failed to write data to key.pem"))
	}

	newCertificate := cloudflare.OriginCACertificate{Hostnames: configValues.Hostnames, RequestType: "origin-rsa", RequestValidity: configValues.Validity, CSR: certOutBuffer.String()}

	api, err := cloudflare.NewWithUserServiceKey(configValues.AuthServiceKey)

	Check(err)
	responseCertificate, err := api.CreateOriginCertificate(newCertificate)
	Check(err)

	certificateFile := filepath.Join(cfbotLiveFolderPath, "certificate.pem")
	certOut, err := os.OpenFile(certificateFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	Check(err)

	//write the certificate we got to the file
	certOut.WriteString(responseCertificate.Certificate)

	//always run the post renew command
	executePostRenewCommnd()

	//if it is an initial run save configs
	if initialRun {
		//after writing the certificate to the file, save the certificate Id in case of initial run
		certificateID := responseCertificate.ID
		configValues.PreviousCertificateID = certificateID
		saveConfigsJSON()
		addCron()
		return
	}
	revokePreviousCertificate()
}

func checkValidityOfCertificate() bool {
	certificateFile := filepath.Join(cfbotLiveFolderPath, "certificate.pem")
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
func verifyDirectoryExists(directoryPath string) {
	_, err := os.Stat(directoryPath)
	if os.IsNotExist(err) && initialRun {
		//if folder does not exist and it is an initial run create a new folder
		err := os.MkdirAll(directoryPath, 0755)
		Check(err)
		return
	} else if err != nil {
		Check(errors.New("Error necessary folders are not setup, if this is the first time running the script please run --init"))
	}
}

//checkInitialRunAndCreateCertificate is used to complete the whole process of verifying existing directory and getting a new certificate
func checkInitialRunAndCreateCertificate() {
	verifyDirectoryExists(cfbotFolderPath)
	verifyDirectoryExists(cfbotLiveFolderPath)
	if initialRun {
		//this means you're running it for the first time, save the config values
		generateCertificate()
		return
	}
	//this usually means you're supposed to get new certificates since the old ones are about to expire
	//TODO:check if certificates need to be renewed && backup existing certificates && then try to get new certificates
	certificatesNeedsRenewal := checkValidityOfCertificate()
	if certificatesNeedsRenewal {
		generateCertificate()
	} else {
		fmt.Printf("%s Certificates are still valid within the buffer time, not getting new certificates\n", warningFlag)
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
	// fmt.Println(strings.Repeat("-", 100))
	authServiceKey := viper.GetString("auth")
	hosts := viper.GetStringSlice("hostnames")
	postRenew := viper.GetString("postRenew")
	validity := viper.GetInt("validity")
	initialRun = viper.GetBool("init")
	//If it is not the first time this is being run, get the certificate Id and add to configvalues
	if !initialRun {
		previousCertificateID := viper.GetString("previousCertificateId")
		configValues.PreviousCertificateID = previousCertificateID
	}
	configValues.AuthServiceKey = authServiceKey
	configValues.Hostnames = hosts
	configValues.Validity = validity
	configValues.PostRenewCommand = postRenew
	validateFlags()
}

//CheckSudo is used to check if the user executing is root or not
func CheckSudo() bool {
	rootUser := os.Geteuid()
	if rootUser != 0 {
		return false
	}
	return true
}
