package cmd

import (
	"fmt"

	"github.com/lf-edge/eden/pkg/openevec"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/thediveo/enumflag"
)

func newDisksCmd() *cobra.Command {
	var disksCmd = &cobra.Command{
		Use:   "disks",
		Short: `Manage disks of edge-node`,
	}

	groups := CommandGroups{
		{
			Message: "Basic Commands",
			Commands: []*cobra.Command{
				newDisksLayoutCmd(),
				newSetDisksLayoutCmd(),
			},
		},
	}

	groups.AddTo(disksCmd)

	return disksCmd
}

func newDisksLayoutCmd() *cobra.Command {
	var getDisksLayoutCmd = &cobra.Command{
		Use:   "get",
		Short: "Get disks layout",
		Long:  `Get disks layout`,
		Run: func(cmd *cobra.Command, args []string) {
			if layout, err := openEVEC.GetDisksLayout(); err != nil {
				log.Fatal(err)
			} else {
				fmt.Println(layout)
			}
		},
	}
	return getDisksLayoutCmd
}

func newSetDisksLayoutCmd() *cobra.Command {
	dc := &openevec.DisksConfig{}

	var setDisksLayoutCmd = &cobra.Command{
		Use:   "set",
		Short: "Set disks layout",
		Long:  `Set disks layout`,
		Run: func(cmd *cobra.Command, args []string) {
			if err := openEVEC.SetDiskLayout(dc); err != nil {
				log.Fatal(err)
			}
		},
	}

	setDisksLayoutCmd.Flags().Var(
		enumflag.New(&dc.LayoutType, "layout-type", openevec.LayoutTypeIds, enumflag.EnumCaseInsensitive),
		"layout-type",
		"sets layout type; can be 'unspecified', 'raid1', 'raid10'")
	setDisksLayoutCmd.Flags().Var(
		enumflag.New(&dc.DiskType, "disk-type", openevec.DiskTypeIds, enumflag.EnumCaseInsensitive),
		"disk-type",
		"sets disk type; can be 'sata', 'virtio', 'nvme'")
	setDisksLayoutCmd.Flags().UintSliceVar(&dc.OfflineDisks, "offline-disks", nil, "list of indexes of offline disks started with 0")
	setDisksLayoutCmd.Flags().UintSliceVar(&dc.UnusedDisks, "unused-disks", nil, "list of indexes of unused disks started with 0")
	setDisksLayoutCmd.Flags().UintSliceVar(&dc.ReplaceDisks, "replace-disks", nil, "list of indexes of replace disks started with 0")
	setDisksLayoutCmd.Flags().UintSliceVar(&dc.PartDisks, "part-disks", nil, "list of indexes of disks to use part only started with 0")

	return setDisksLayoutCmd
}
