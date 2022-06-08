package cmd

import (
	"fmt"

	"github.com/lf-edge/eden/pkg/device"
	"github.com/lf-edge/eden/pkg/utils"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/thediveo/enumflag"
)

var (
	layoutType   device.DisksLayoutType
	diskType     device.DiskType
	offlineDisks []uint
	unusedDisks  []uint
	replaceDisks []uint
	partDisks    []uint
)

var diskTypeIds = map[device.DiskType][]string{
	device.DiskTypeSata:   {"sata"},
	device.DiskTypeVirtio: {"virtio"},
	device.DiskTypeNVME:   {"nvme"},
}

var layoutTypeIds = map[device.DisksLayoutType][]string{
	device.DisksLayoutTypeUnspecified: {"unspecified"},
	device.DisksLayoutTypeRaid1:       {"raid1"},
	device.DisksLayoutTypeRaid10:      {"raid10"},
}

var disksCmd = &cobra.Command{
	Use:   "disks",
	Short: `Manage disks of edge-node`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if err := rootCmd.PersistentPreRunE(cmd, args); err != nil {
			return err
		}
		assignCobraToViper(cmd)

		_, err := utils.LoadConfigFile(configFile)
		if err != nil {
			return err
		}
		return nil
	},
}

var getDisksLayoutCmd = &cobra.Command{
	Use:   "get",
	Short: "Get disks layout",
	Long:  `Get disks layout`,
	Run: func(cmd *cobra.Command, args []string) {
		changer := &adamChanger{}
		_, dev, err := changer.getControllerAndDev()
		if err != nil {
			log.Fatalf("getControllerAndDev error: %s", err)
		}
		layout := dev.GetDiskLayout()
		fmt.Println(layout) //nolint:forbidigo
	},
}

var setDisksLayoutCmd = &cobra.Command{
	Use:   "set",
	Short: "Set disks layout",
	Long:  `Set disks layout`,
	Run: func(cmd *cobra.Command, args []string) {
		changer := &adamChanger{}
		ctrl, dev, err := changer.getControllerAndDev()
		if err != nil {
			log.Fatalf("getControllerAndDev error: %s", err)
		}
		layout := dev.GetDiskLayout()
		if layout == nil {
			layout = &device.DisksLayout{}
		}
		layout.LayoutType = layoutType
		layout.DiskType = diskType
		layout.OfflineDisks = offlineDisks
		layout.UnusedDisks = unusedDisks
		layout.ReplaceDisks = replaceDisks
		layout.PartDisks = partDisks
		dev.SetDiskLayout(layout)
		if err = changer.setControllerAndDev(ctrl, dev); err != nil {
			log.Fatalf("setControllerAndDev: %s", err)
		}
	},
}

func disksInit() {
	disksCmd.AddCommand(getDisksLayoutCmd)
	setDisksLayoutCmd.Flags().Var(
		enumflag.New(&layoutType, "layout-type", layoutTypeIds, enumflag.EnumCaseInsensitive),
		"layout-type",
		"sets layout type; can be 'unspecified', 'raid1', 'raid10'")
	setDisksLayoutCmd.Flags().Var(
		enumflag.New(&diskType, "disk-type", diskTypeIds, enumflag.EnumCaseInsensitive),
		"disk-type",
		"sets disk type; can be 'sata', 'virtio', 'nvme'")
	setDisksLayoutCmd.Flags().UintSliceVar(&offlineDisks, "offline-disks", nil, "list of indexes of offline disks started with 0")
	setDisksLayoutCmd.Flags().UintSliceVar(&unusedDisks, "unused-disks", nil, "list of indexes of unused disks started with 0")
	setDisksLayoutCmd.Flags().UintSliceVar(&replaceDisks, "replace-disks", nil, "list of indexes of replace disks started with 0")
	setDisksLayoutCmd.Flags().UintSliceVar(&partDisks, "part-disks", nil, "list of indexes of disks to use part only started with 0")
	disksCmd.AddCommand(setDisksLayoutCmd)
}
