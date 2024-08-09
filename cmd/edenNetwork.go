package cmd

import (
	"github.com/lf-edge/eden/pkg/controller/types"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/thediveo/enumflag"
)

func newNetworkCmd() *cobra.Command {
	var networkCmd = &cobra.Command{
		Use: "network",
	}

	groups := CommandGroups{
		{
			Message: "Basic Commands",
			Commands: []*cobra.Command{
				newNetworkLsCmd(),
				newNetworkDeleteCmd(),
				newNetworkNetstatCmd(),
				newNetworkCreateCmd(),
			},
		},
	}

	groups.AddTo(networkCmd)

	return networkCmd
}

func newNetworkLsCmd() *cobra.Command {
	var outputFormat types.OutputFormat
	//networkLsCmd is a command to list deployed network instances
	var networkLsCmd = &cobra.Command{
		Use:   "ls",
		Short: "List networks",
		Run: func(cmd *cobra.Command, args []string) {
			if err := openEVEC.NetworkLs(outputFormat); err != nil {
				log.Fatal(err)
			}
		},
	}
	networkLsCmd.Flags().Var(
		enumflag.New(&outputFormat, "format", outputFormatIds, enumflag.EnumCaseInsensitive),
		"format",
		"Format to print logs, supports: lines, json")
	return networkLsCmd
}

func newNetworkDeleteCmd() *cobra.Command {
	//networkDeleteCmd is a command to delete network instance from EVE
	var networkDeleteCmd = &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete network",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			niName := args[0]
			if err := openEVEC.NetworkDelete(niName); err != nil {
				log.Fatal(err)
			}
		},
	}
	return networkDeleteCmd
}

func newNetworkNetstatCmd() *cobra.Command {
	var outputTail uint
	var outputFormat types.OutputFormat

	//networkNetstatCmd is a command to show netstat for network
	var networkNetstatCmd = &cobra.Command{
		Use:   "netstat <name>",
		Short: "Show netstat for network",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			niName := args[0]
			if err := openEVEC.NetworkNetstat(niName, outputFormat, outputTail); err != nil {
				log.Fatal(err)
			}
		},
	}

	networkNetstatCmd.Flags().UintVar(&outputTail, "tail", 0, "Show only last N lines")
	networkNetstatCmd.Flags().Var(
		enumflag.New(&outputFormat, "format", outputFormatIds, enumflag.EnumCaseInsensitive),
		"format",
		"Format to print logs, supports: lines, json")

	return networkNetstatCmd
}

func newNetworkCreateCmd() *cobra.Command {
	var networkType, networkName, uplinkAdapter string
	var staticDNSEntries []string
	var enableFlowlog bool

	//networkCreateCmd is command for create network instance in EVE
	var networkCreateCmd = &cobra.Command{
		Use:   "create [subnet]",
		Short: "Create network instance in EVE",
		Args:  cobra.RangeArgs(0, 1),
		Run: func(cmd *cobra.Command, args []string) {
			subnet := ""
			if len(args) == 1 {
				subnet = args[0]
			}
			if err := openEVEC.NetworkCreate(subnet, networkType, networkName, uplinkAdapter,
				staticDNSEntries, enableFlowlog); err != nil {
				log.Fatal(err)
			}
		},
	}

	networkCreateCmd.Flags().StringVar(&networkType, "type", "local", "Type of network: local or switch")
	networkCreateCmd.Flags().StringVarP(&networkName, "name", "n", "", "Name of network (empty for auto generation)")
	networkCreateCmd.Flags().StringVarP(&uplinkAdapter, "uplink", "u", "eth0", "Name of uplink adapter, set to 'none' to not use uplink")
	networkCreateCmd.Flags().StringArrayVarP(&staticDNSEntries, "static-dns-entries", "s", []string{}, "List of static DNS entries in format HOSTNAME:IP_ADDR,IP_ADDR,...")
	networkCreateCmd.Flags().BoolVar(&enableFlowlog, "enable-flowlog", false, "enable flow logging (EVE collecting and publishing records of application network flows)")

	return networkCreateCmd
}
