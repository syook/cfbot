package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "cfbot",
	Short: "Automatically get/renew new certificates from Cloudflare",
	Long:  `cfbot is a CLI application for cloudflare that helps you automate getting certificates from cloudflare.`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	//	Run: func(cmd *cobra.Command, args []string) { },
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
	cobra.OnInitialize(func() {
		initConfig()
		rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.cfbot.yaml)")
		rootCmd.PersistentFlags().String("auth", "", "Origin CA key to be used as auth")
		rootCmd.MarkPersistentFlagRequired("auth")

		// viper.BindPFlag("auth", rootCmd.PersistentFlags().Lookup("auth"))

		// Cobra also supports local flags, which will only run
		// when this action is called directly.
		rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
		postInitCommands(rootCmd.Commands())
	})
}

func postInitCommands(commands []*cobra.Command) {
	for _, cmd := range commands {
		presetRequiredFlags(cmd)
		if cmd.HasSubCommands() {
			postInitCommands(cmd.Commands())
		}
	}
}

func presetRequiredFlags(cmd *cobra.Command) {
	viper.BindPFlags(cmd.Flags())
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		if viper.IsSet(f.Name) && viper.GetString(f.Name) != "" {
			cmd.Flags().Set(f.Name, viper.GetString(f.Name))
		}
	})
}

// func init() {
// cobra.OnInitialize(initConfig)

// Here you will define your flags and configuration settings.
// Cobra supports persistent flags, which, if defined here,
// will be global for your application.

// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.cfbot.yaml)")
// rootCmd.PersistentFlags().String("auth", "", "Origin CA key to be used as auth")
// rootCmd.MarkPersistentFlagRequired("auth")
// viper.BindPFlag("auth", rootCmd.PersistentFlags().Lookup("auth"))
// // Cobra also supports local flags, which will only run
// // when this action is called directly.
// rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
// }

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		//TODO:
		//By default now adding certs and creating a directory for cfbot at the home directory level, need to make this dynamic
		// Find home directory.
		home, err := homedir.Dir()
		certsdir := filepath.Join(home, "certs")

		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Search config in home/certs directory with name cfbot.json
		viper.SetConfigType("json")
		viper.SetConfigFile("cfbot.json")
		viper.AddConfigPath(certsdir)
	}

	// viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	// if err := viper.ReadInConfig(); err == nil {
	// 	fmt.Println("Using config file:", viper.ConfigFileUsed())
	// }

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found; ignore error if desired
			fmt.Println("file not found")
		} else {
			fmt.Println("some error", err)
			// Config file was found but another error was produced
		}
	}
	fmt.Println("Config file used", viper.ConfigFileUsed())
}
