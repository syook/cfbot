package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/syook/cfbot/structs"
	"github.com/syook/cfbot/utils"
)

// getCmd represents the get command
var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get New Certificates",
	Long:  `Get New Certificates from cloudflare.`,
	Run: func(cmd *cobra.Command, args []string) {
		destination := viper.GetString("destination")
		authServiceKey := viper.GetString("auth")
		hosts := viper.GetStringSlice("hostnames")
		validity := viper.GetInt("validity")
		// fmt.Println(authServiceKey, hosts, validity)
		configValues := structs.Configs{AuthServiceKey: authServiceKey, Hostnames: hosts, Validity: validity}
		// utils.CheckValuesAndCreateCertificate(configValues)
		utils.ValidateFlags(configValues, destination)
	},
}

func init() {
	rootCmd.AddCommand(getCmd)

	// Here you will define your flags and configuration settings.
	getCmd.Flags().StringSlice("hostnames", []string{}, "Hostnames for SAN")
	viper.BindPFlag("hostnames", getCmd.Flags().Lookup("hostnames"))
	getCmd.Flags().IntP("validity", "v", 30, "Validity for the certificates")
	viper.BindPFlag("validity", getCmd.Flags().Lookup("validity"))
}
