package openevec

import (
	"fmt"

	"github.com/lf-edge/eden/pkg/device"
)

type DisksConfig struct {
	LayoutType   device.DisksLayoutType
	DiskType     device.DiskType
	OfflineDisks []uint
	UnusedDisks  []uint
	ReplaceDisks []uint
	PartDisks    []uint
}

var DiskTypeIds = map[device.DiskType][]string{
	device.DiskTypeSata:   {"sata"},
	device.DiskTypeVirtio: {"virtio"},
	device.DiskTypeNVME:   {"nvme"},
}

var LayoutTypeIds = map[device.DisksLayoutType][]string{
	device.DisksLayoutTypeUnspecified: {"unspecified"},
	device.DisksLayoutTypeRaid1:       {"raid1"},
	device.DisksLayoutTypeRaid10:      {"raid10"},
}

func GetDisksLayout() (device.DisksLayout, error) {
	changer := &adamChanger{}
	_, dev, err := changer.getControllerAndDev()
	if err != nil {
		return device.DisksLayout{}, fmt.Errorf("getControllerAndDev error: %w", err)
	}
	layout := dev.GetDiskLayout()
	return *layout, nil
}

func SetDiskLayout(dc *DisksConfig) error {
	changer := &adamChanger{}
	ctrl, dev, err := changer.getControllerAndDev()
	if err != nil {
		return fmt.Errorf("getControllerAndDev error: %w", err)
	}
	layout := dev.GetDiskLayout()
	if layout == nil {
		layout = &device.DisksLayout{}
	}
	layout.LayoutType = dc.LayoutType
	layout.DiskType = dc.DiskType
	layout.OfflineDisks = dc.OfflineDisks
	layout.UnusedDisks = dc.UnusedDisks
	layout.ReplaceDisks = dc.ReplaceDisks
	layout.PartDisks = dc.PartDisks
	dev.SetDiskLayout(layout)
	if err = changer.setControllerAndDev(ctrl, dev); err != nil {
		return fmt.Errorf("setControllerAndDev: %w", err)
	}
	return nil
}
