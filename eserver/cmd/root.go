package cmd

import (
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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
	_ = rootCmd.Execute()
}
