package cmd

import (
	"github.com/spf13/cobra"
)

var utilsCmd = &cobra.Command{
	Use:   "utils",
	Short: "Eden utilities",
	Long: `Additional utilities for EDEN.`,
}

func utilsInit() {
	utilsCmd.AddCommand(templateCmd)	
}
