package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"strings"
)

var rootCmd = &cobra.Command{Use: "eserver"}

func init() {
	viper.SetEnvPrefix("eserver")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()

	rootCmd.AddCommand(serverCmd)
	serverInit()
}

// Execute primary function for cobra
func Execute() {
	rootCmd.Execute()
}
