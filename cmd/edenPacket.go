package cmd

import (
	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/openevec"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func newPacketCmd(configName, verbosity *string) *cobra.Command {
	cfg := &openevec.EdenSetupArgs{}
	var packetProjectName string

	var packetCmd = &cobra.Command{
		Use:   "packet",
		Short: `Manage VMs in Equinix Metal Platform`,
	}

	packetCmd.AddCommand(newPacketVMCmd(cfg, &cfg.Packet.Key, &packetProjectName))

	packetCmd.PersistentFlags().StringVarP(&cfg.Packet.Key, "key", "k", "", "packet key file")
	packetCmd.PersistentFlags().StringVarP(&packetProjectName, "project", "p", defaults.DefaultPacketProjectName, "project name on packet")

	return packetCmd
}

func newPacketVMCmd(cfg *openevec.EdenSetupArgs, packetKey, packetProjectName *string) *cobra.Command {
	var packetVMCmd = &cobra.Command{
		Use:   "vm",
		Short: `Manage VMs in packet`,
	}

	groups := CommandGroups{
		{
			Message: "Basic Commands",
			Commands: []*cobra.Command{
				newPacketRunCmd(cfg, packetKey, packetProjectName),
				newPacketDeleteCmd(packetKey, packetProjectName),
				newPacketGetIPCmd(packetKey, packetProjectName),
			},
		},
	}

	groups.AddTo(packetVMCmd)

	return packetVMCmd
}

func newPacketRunCmd(cfg *openevec.EdenSetupArgs, packetKey, packetProjectName *string) *cobra.Command {
	var packetVMName, packetZone, packetMachineType, packetIPXEUrl string

	var packetRun = &cobra.Command{
		Use:   "run",
		Short: "run vm in packet",
		Run: func(cmd *cobra.Command, args []string) {
			err := openevec.PacketRun(*packetKey, *packetProjectName, packetVMName, packetZone, packetMachineType, packetIPXEUrl, cfg)
			if err != nil {
				log.Fatal(err)
			}
		},
	}

	packetRun.Flags().StringVar(&packetVMName, "vm-name", defaults.DefaultVMName, "vm name")
	packetRun.Flags().StringVar(&packetZone, "zone", defaults.DefaultPacketZone, "packet zone")
	packetRun.Flags().StringVar(&packetMachineType, "machine-type", defaults.DefaultPacketMachineType, "packet machine type")
	packetRun.Flags().StringVar(&packetIPXEUrl, "ipxe-url", "", "packet ipxe url")

	return packetRun
}

func newPacketDeleteCmd(packetKey, packetProjectName *string) *cobra.Command {
	var packetVMName string

	var packetDelete = &cobra.Command{
		Use:   "delete",
		Short: "delete vm from packet",
		Run: func(cmd *cobra.Command, args []string) {
			if err := openevec.PacketDelete(*packetKey, *packetProjectName, packetVMName); err != nil {
				log.Fatal(err)
			}
		},
	}

	packetDelete.Flags().StringVar(&packetVMName, "vm-name", defaults.DefaultVMName, "vm name")

	return packetDelete
}

func newPacketGetIPCmd(packetKey, packetProjectName *string) *cobra.Command {
	var packetVMName string

	var packetGetIP = &cobra.Command{
		Use:   "get-ip",
		Short: "print IP of VM in packet",
		Run: func(cmd *cobra.Command, args []string) {
			if err := openevec.PacketGetIP(*packetKey, *packetProjectName, packetVMName); err != nil {
				log.Fatal(err)
			}
		},
	}

	packetGetIP.Flags().StringVar(&packetVMName, "vm-name", defaults.DefaultVMName, "vm name")

	return packetGetIP
}
