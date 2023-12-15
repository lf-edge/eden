package device

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/lf-edge/eve-api/go/config"
	"github.com/lf-edge/eve-api/go/evecommon"
)

// DisksLayoutType stores expectation about disks layout
type DisksLayoutType int

// DisksLayoutType enum
const (
	DisksLayoutTypeUnspecified DisksLayoutType = iota // no configured
	DisksLayoutTypeRaid1                              // mirror (2 disks)
	DisksLayoutTypeRaid10                             // striped mirrors (4 disks)
)

func (layoutType DisksLayoutType) maxDisks() uint {
	switch layoutType {
	case DisksLayoutTypeRaid1:
		return 2
	case DisksLayoutTypeRaid10:
		return 4
	case DisksLayoutTypeUnspecified:
		return 0
	default:
		return 0
	}
}

// DiskType stores expectation about disk type
type DiskType int

// DiskType enum
const (
	DiskTypeSata   DiskType = iota // sda
	DiskTypeNVME                   // nvme0n1
	DiskTypeVirtio                 // vda
)

const partNumber = "9" // predefined partition with zfs

// DisksLayout stores data for disks layout preparation
type DisksLayout struct {
	DiskType     DiskType // to calculate name based on index
	LayoutType   DisksLayoutType
	OfflineDisks []uint // indexes of offline disks
	UnusedDisks  []uint // indexes of unused disks
	ReplaceDisks []uint // indexes of disks to be replaced. Replacements will be selected from disks not in use
	PartDisks    []uint // indexes of disks to use partition in name
}

func (diskType DiskType) getName(layout *DisksLayout, ind uint) string {
	name := ""
	switch diskType {
	case DiskTypeSata:
		name = fmt.Sprintf("/dev/sd%c", rune(uint('a')+ind))
	case DiskTypeNVME:
		name = fmt.Sprintf("/dev/nvme%dn1", ind)
	case DiskTypeVirtio:
		name = fmt.Sprintf("/dev/vd%c", rune(uint('a')+ind))
	}
	for _, el := range layout.PartDisks {
		if el == ind {
			if diskType == DiskTypeNVME {
				name += "p"
			}
			name += partNumber
		}
	}

	return name
}

func getDiskTypeIndexAndPart(name string) (DiskType, uint, bool, error) {
	isPart := false
	if strings.HasSuffix(name, partNumber) {
		isPart = true
	}
	if strings.HasPrefix(name, "/dev/sd") {
		runes := []rune(strings.TrimPrefix(strings.TrimSuffix(name, partNumber), "/dev/sd"))
		if len(runes) == 0 {
			return DiskTypeSata, 0, isPart, fmt.Errorf("cannot extract index from disk name: %s", name)
		}
		return DiskTypeSata, uint(runes[0]) - uint('a'), isPart, nil
	}
	if strings.HasPrefix(name, "/dev/vd") {
		runes := []rune(strings.TrimPrefix(strings.TrimSuffix(name, partNumber), "/dev/vd"))
		if len(runes) == 0 {
			return DiskTypeSata, 0, isPart, fmt.Errorf("cannot extract index from disk name: %s", name)
		}
		return DiskTypeVirtio, uint(runes[0]) - uint('a'), isPart, nil
	}
	if strings.HasPrefix(name, "/dev/nvme") {
		ind, err := strconv.Atoi(strings.TrimSuffix(strings.TrimPrefix(strings.TrimSuffix(name, "p"+partNumber), "/dev/nvme"), "n1"))
		if err != nil {
			return DiskTypeSata, 0, isPart, fmt.Errorf("cannot extract index from disk name %s: %w", name, err)
		}
		return DiskTypeNVME, uint(ind), isPart, nil
	}
	return DiskTypeSata, 0, isPart, fmt.Errorf("unexpected disk name: %s", name)
}

func (layout *DisksLayout) getDiskState(ind uint) config.DiskConfigType {
	diskConfigState := config.DiskConfigType_DISK_CONFIG_TYPE_ZFS_ONLINE
	for _, el := range layout.OfflineDisks {
		if el == ind {
			diskConfigState = config.DiskConfigType_DISK_CONFIG_TYPE_ZFS_OFFLINE
			break
		}
	}
	for _, el := range layout.UnusedDisks {
		if el == ind {
			diskConfigState = config.DiskConfigType_DISK_CONFIG_TYPE_UNUSED
			break
		}
	}
	return diskConfigState
}

// getDisk returns disk config based on provided disk index
func (layout *DisksLayout) getDisk(ind uint) *config.DiskConfig {
	name := layout.DiskType.getName(layout, ind)
	cfg := &config.DiskConfig{
		Disk: &evecommon.DiskDescription{
			Name: name,
		},
		DiskConfig: layout.getDiskState(ind),
	}
	for i, el := range layout.ReplaceDisks {
		if el == ind {
			cfg.OldDisk = cfg.Disk
			// select next disk on top of max disks
			newInd := layout.LayoutType.maxDisks() + uint(i)
			cfg.Disk = &evecommon.DiskDescription{
				Name: layout.DiskType.getName(layout, newInd),
			}
			break
		}
	}
	return cfg
}

// GetDisksConfig returns disks config based on layout
func (layout *DisksLayout) GetDisksConfig() (*config.DisksConfig, error) {
	if layout == nil {
		return nil, errors.New("nil layout provided")
	}
	var disksConfig config.DisksConfig
	switch layout.LayoutType {
	case DisksLayoutTypeUnspecified:
		// nothing to configure
	case DisksLayoutTypeRaid1:
		disksConfig.Children = append(disksConfig.Children,
			&config.DisksConfig{
				ArrayType: config.DisksArrayType_DISKS_ARRAY_TYPE_RAID1,
				Disks: []*config.DiskConfig{
					layout.getDisk(0),
					layout.getDisk(1),
				},
			},
		)
		disksConfig.ArrayType = config.DisksArrayType_DISKS_ARRAY_TYPE_RAID0
	case DisksLayoutTypeRaid10:
		disksConfig.Children = append(disksConfig.Children,
			&config.DisksConfig{
				ArrayType: config.DisksArrayType_DISKS_ARRAY_TYPE_RAID1,
				Disks: []*config.DiskConfig{
					layout.getDisk(0),
					layout.getDisk(1),
				},
			},
		)
		disksConfig.Children = append(disksConfig.Children,
			&config.DisksConfig{
				ArrayType: config.DisksArrayType_DISKS_ARRAY_TYPE_RAID1,
				Disks: []*config.DiskConfig{
					layout.getDisk(2),
					layout.getDisk(3),
				},
			},
		)
		disksConfig.ArrayType = config.DisksArrayType_DISKS_ARRAY_TYPE_RAID0
	default:
		return nil, fmt.Errorf("not implemented disks layout: %d", layout.LayoutType)
	}
	return &disksConfig, nil
}

// ParseDiskLayout from configuration
//
//nolint:cyclop
func ParseDiskLayout(disksConfig *config.DisksConfig) (*DisksLayout, error) {
	if disksConfig == nil {
		return nil, errors.New("nil disksConfig provided")
	}
	disksLayout := DisksLayout{}
	switch disksConfig.ArrayType {
	case config.DisksArrayType_DISKS_ARRAY_TYPE_RAID0:
		switch len(disksConfig.Children) {
		case 1:
			disksLayout.LayoutType = DisksLayoutTypeRaid1
		case 2:
			disksLayout.LayoutType = DisksLayoutTypeRaid10
		default:
			return nil, fmt.Errorf("unexpected children count: %d", len(disksConfig.Children))
		}
		for _, el := range disksConfig.Children {
			for _, disk := range el.Disks {
				diskType, diskIndex, part, err := getDiskTypeIndexAndPart(disk.Disk.Name)
				if err != nil {
					return nil, err
				}
				if part {
					disksLayout.PartDisks = append(disksLayout.PartDisks, diskIndex)
				}
				if disk.OldDisk != nil {
					diskType, diskIndex, part, err = getDiskTypeIndexAndPart(disk.OldDisk.Name)
					if err != nil {
						return nil, err
					}
					disksLayout.ReplaceDisks = append(disksLayout.ReplaceDisks, diskIndex)
					if part {
						disksLayout.PartDisks = append(disksLayout.PartDisks, diskIndex)
					}
				}
				disksLayout.DiskType = diskType
				if disk.DiskConfig == config.DiskConfigType_DISK_CONFIG_TYPE_UNUSED {
					disksLayout.UnusedDisks = append(disksLayout.UnusedDisks, diskIndex)
				}
				// we expect now that unused disks also present in OfflineDisks
				if disk.DiskConfig == config.DiskConfigType_DISK_CONFIG_TYPE_ZFS_OFFLINE || disk.DiskConfig == config.DiskConfigType_DISK_CONFIG_TYPE_UNUSED {
					disksLayout.OfflineDisks = append(disksLayout.OfflineDisks, diskIndex)
				}
			}
		}
	case config.DisksArrayType_DISKS_ARRAY_TYPE_UNSPECIFIED:
		// nothing to process
	case config.DisksArrayType_DISKS_ARRAY_TYPE_RAID1, config.DisksArrayType_DISKS_ARRAY_TYPE_RAID5, config.DisksArrayType_DISKS_ARRAY_TYPE_RAID6:
		return nil, fmt.Errorf("unsupported ArrayType: %s", disksConfig.ArrayType)
	default:
		return nil, fmt.Errorf("unexpected ArrayType: %s", disksConfig.ArrayType)
	}
	return &disksLayout, nil
}

// String returns string representation of disks layout
func (layout *DisksLayout) String() string {
	if layout == nil {
		return ""
	}
	data, err := json.Marshal(layout)
	if err != nil {
		return ""
	}
	return string(data)
}
