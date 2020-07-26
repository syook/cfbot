package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/syook/cfbot/utils"

	"github.com/spf13/viper"
)

// var destination string
var initialRun bool

const cfbotFilePath string = "/etc/cfbot"
const version string = "0.4.0"

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:     "cfbot",
	Short:   "Automatically get/renew new certificates from Cloudflare",
	Long:    `CFbot is a CLI application for cloudflare that helps you automate getting certificates from cloudflare.`,
	Version: version,
	Run: func(cmd *cobra.Command, args []string) {
		utils.Cfbot()
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	//allow the users to run this script only as sudo, because of the permissions needed to add the cron jobs and also to store the certs in /etc/cfbot
	if !utils.CheckSudo() {
		utils.Check(errors.New("Please Run as root. (Sudo)"))
	}
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.
	rootCmd.PersistentFlags().BoolVar(&initialRun, "init", false, "Initialize the service")
	viper.BindPFlag("init", rootCmd.PersistentFlags().Lookup("init"))
	// rootCmd.PersistentFlags().StringVarP(&destination, "destination", "d", "", "destination directory to save certs files (default is $HOME/certs)")
	// viper.BindPFlag("destination", rootCmd.PersistentFlags().Lookup("destination"))
	rootCmd.PersistentFlags().String("auth", "", "Origin CA key to be used as auth")
	viper.BindPFlag("auth", rootCmd.PersistentFlags().Lookup("auth"))
	rootCmd.Flags().StringSlice("hostnames", []string{}, "Hostnames for SAN")
	viper.BindPFlag("hostnames", rootCmd.Flags().Lookup("hostnames"))
	rootCmd.Flags().IntP("validity", "v", 30, "Validity for the certificates")
	viper.BindPFlag("validity", rootCmd.Flags().Lookup("validity"))
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {

	if initialRun {
		return
	}
	viper.SetConfigType("json")
	configFile := filepath.Join(cfbotFilePath, "cfbot.json")
	viper.SetConfigFile(configFile)

	viper.AutomaticEnv() // read in environment variables that match
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found; ignore error if desired
			fmt.Println("file not found")
			utils.Check(err)
		} else {
			// Config file was found but another error was produced
			fmt.Println("some error while", err)
			utils.Check(err)
		}
	}
	fmt.Println("config file used", viper.ConfigFileUsed())
}
