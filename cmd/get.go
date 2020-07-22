package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/syook/cfbot/utils"
)

// getCmd represents the get command
var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get new Certificates",
	Long:  `Get new Certificates from cloudflare.`,
	Run: func(cmd *cobra.Command, args []string) {
		authServiceKey := viper.GetString("auth")  //cmd.Parent().PersistentFlags().GetString("auth")
		hosts := viper.GetStringSlice("hostnames") //cmd.Flags().GetStringSlice("hostnames")
		validity := viper.GetInt("validity")       //cmd.Flags().GetInt("validity")
		// fmt.Println(authServiceKey, hosts, validity)
		utils.CheckValuesAndCreateCertificate(authServiceKey, hosts, validity)
		// utils.SaveConfigs(authServiceKey, hosts, validity)
		// utils.VerifyDirectoryExists()
		// utils.GenerateCertificate(authServiceKey, hosts, validity)
	},
}

func init() {
	rootCmd.AddCommand(getCmd)

	// Here you will define your flags and configuration settings.
	getCmd.Flags().StringSlice("hostnames", []string{}, "Hostnames for SAN")
	// viper.BindPFlag("hostnames", getCmd.Flags().Lookup("hostnames"))
	getCmd.Flags().IntP("validity", "v", 30, "Validity for the certificates")
	// viper.BindPFlag("validity", getCmd.Flags().Lookup("validity"))
	getCmd.MarkFlagRequired("hostnames")
	getCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
