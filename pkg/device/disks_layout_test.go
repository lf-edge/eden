package device_test

import (
	"encoding/json"
	"testing"

	"github.com/lf-edge/eden/pkg/device"
	"github.com/lf-edge/eve-api/go/config"
	"github.com/lf-edge/eve-api/go/evecommon"
	"github.com/stretchr/testify/assert"
)

func TestConversionLayout(t *testing.T) {
	t.Parallel()

	testMatrix := map[string]struct {
		layout      *device.DisksLayout
		disksConfig *config.DisksConfig
	}{
		"raid1": {
			layout: &device.DisksLayout{
				DiskType:     device.DiskTypeSata,
				LayoutType:   device.DisksLayoutTypeRaid1,
				OfflineDisks: nil,
				UnusedDisks:  nil,
				ReplaceDisks: nil,
			},
			disksConfig: &config.DisksConfig{
				ArrayType: config.DisksArrayType_DISKS_ARRAY_TYPE_RAID0,
				Children: []*config.DisksConfig{
					{
						Disks: []*config.DiskConfig{
							{
								Disk: &evecommon.DiskDescription{
									Name: "/dev/sda",
								},
								DiskConfig: config.DiskConfigType_DISK_CONFIG_TYPE_ZFS_ONLINE,
							},
							{
								Disk: &evecommon.DiskDescription{
									Name: "/dev/sdb",
								},
								DiskConfig: config.DiskConfigType_DISK_CONFIG_TYPE_ZFS_ONLINE,
							},
						},
						ArrayType: config.DisksArrayType_DISKS_ARRAY_TYPE_RAID1,
					},
				},
			},
		},
		"raid1-part": {
			layout: &device.DisksLayout{
				DiskType:     device.DiskTypeSata,
				LayoutType:   device.DisksLayoutTypeRaid1,
				OfflineDisks: nil,
				UnusedDisks:  nil,
				ReplaceDisks: nil,
				PartDisks:    []uint{0},
			},
			disksConfig: &config.DisksConfig{
				ArrayType: config.DisksArrayType_DISKS_ARRAY_TYPE_RAID0,
				Children: []*config.DisksConfig{
					{
						Disks: []*config.DiskConfig{
							{
								Disk: &evecommon.DiskDescription{
									Name: "/dev/sda9",
								},
								DiskConfig: config.DiskConfigType_DISK_CONFIG_TYPE_ZFS_ONLINE,
							},
							{
								Disk: &evecommon.DiskDescription{
									Name: "/dev/sdb",
								},
								DiskConfig: config.DiskConfigType_DISK_CONFIG_TYPE_ZFS_ONLINE,
							},
						},
						ArrayType: config.DisksArrayType_DISKS_ARRAY_TYPE_RAID1,
					},
				},
			},
		},
		"raid10": {
			layout: &device.DisksLayout{
				DiskType:     device.DiskTypeSata,
				LayoutType:   device.DisksLayoutTypeRaid10,
				OfflineDisks: nil,
				UnusedDisks:  nil,
				ReplaceDisks: nil,
			},
			disksConfig: &config.DisksConfig{
				ArrayType: config.DisksArrayType_DISKS_ARRAY_TYPE_RAID0,
				Children: []*config.DisksConfig{
					{
						Disks: []*config.DiskConfig{
							{
								Disk: &evecommon.DiskDescription{
									Name: "/dev/sda",
								},
								DiskConfig: config.DiskConfigType_DISK_CONFIG_TYPE_ZFS_ONLINE,
							},
							{
								Disk: &evecommon.DiskDescription{
									Name: "/dev/sdb",
								},
								DiskConfig: config.DiskConfigType_DISK_CONFIG_TYPE_ZFS_ONLINE,
							},
						},
						ArrayType: config.DisksArrayType_DISKS_ARRAY_TYPE_RAID1,
					}, {
						Disks: []*config.DiskConfig{
							{
								Disk: &evecommon.DiskDescription{
									Name: "/dev/sdc",
								},
								DiskConfig: config.DiskConfigType_DISK_CONFIG_TYPE_ZFS_ONLINE,
							},
							{
								Disk: &evecommon.DiskDescription{
									Name: "/dev/sdd",
								},
								DiskConfig: config.DiskConfigType_DISK_CONFIG_TYPE_ZFS_ONLINE,
							},
						},
						ArrayType: config.DisksArrayType_DISKS_ARRAY_TYPE_RAID1,
					},
				},
			},
		},
		"raid1-offline": {
			layout: &device.DisksLayout{
				DiskType:     device.DiskTypeSata,
				LayoutType:   device.DisksLayoutTypeRaid1,
				OfflineDisks: []uint{1},
				UnusedDisks:  nil,
				ReplaceDisks: nil,
			},
			disksConfig: &config.DisksConfig{
				ArrayType: config.DisksArrayType_DISKS_ARRAY_TYPE_RAID0,
				Children: []*config.DisksConfig{
					{
						Disks: []*config.DiskConfig{
							{
								Disk: &evecommon.DiskDescription{
									Name: "/dev/sda",
								},
								DiskConfig: config.DiskConfigType_DISK_CONFIG_TYPE_ZFS_ONLINE,
							},
							{
								Disk: &evecommon.DiskDescription{
									Name: "/dev/sdb",
								},
								DiskConfig: config.DiskConfigType_DISK_CONFIG_TYPE_ZFS_OFFLINE,
							},
						},
						ArrayType: config.DisksArrayType_DISKS_ARRAY_TYPE_RAID1,
					},
				},
			},
		},
		"raid1-unused": {
			layout: &device.DisksLayout{
				DiskType:     device.DiskTypeSata,
				LayoutType:   device.DisksLayoutTypeRaid1,
				OfflineDisks: []uint{1},
				UnusedDisks:  []uint{1},
				ReplaceDisks: nil,
			},
			disksConfig: &config.DisksConfig{
				ArrayType: config.DisksArrayType_DISKS_ARRAY_TYPE_RAID0,
				Children: []*config.DisksConfig{
					{
						Disks: []*config.DiskConfig{
							{
								Disk: &evecommon.DiskDescription{
									Name: "/dev/sda",
								},
								DiskConfig: config.DiskConfigType_DISK_CONFIG_TYPE_ZFS_ONLINE,
							},
							{
								Disk: &evecommon.DiskDescription{
									Name: "/dev/sdb",
								},
								DiskConfig: config.DiskConfigType_DISK_CONFIG_TYPE_UNUSED,
							},
						},
						ArrayType: config.DisksArrayType_DISKS_ARRAY_TYPE_RAID1,
					},
				},
			},
		},
		"raid1-replace": {
			layout: &device.DisksLayout{
				DiskType:     device.DiskTypeSata,
				LayoutType:   device.DisksLayoutTypeRaid1,
				OfflineDisks: []uint{1},
				UnusedDisks:  []uint{1},
				ReplaceDisks: []uint{1},
			},
			disksConfig: &config.DisksConfig{
				ArrayType: config.DisksArrayType_DISKS_ARRAY_TYPE_RAID0,
				Children: []*config.DisksConfig{
					{
						Disks: []*config.DiskConfig{
							{
								Disk: &evecommon.DiskDescription{
									Name: "/dev/sda",
								},
								DiskConfig: config.DiskConfigType_DISK_CONFIG_TYPE_ZFS_ONLINE,
							},
							{
								Disk: &evecommon.DiskDescription{
									Name: "/dev/sdc",
								},
								OldDisk: &evecommon.DiskDescription{
									Name: "/dev/sdb",
								},
								DiskConfig: config.DiskConfigType_DISK_CONFIG_TYPE_UNUSED,
							},
						},
						ArrayType: config.DisksArrayType_DISKS_ARRAY_TYPE_RAID1,
					},
				},
			},
		},
	}
	for name, test := range testMatrix {
		t.Logf("Running test case %s", name)
		disksConfig, err := test.layout.GetDisksConfig()
		assert.NoError(t, err)
		t.Log(disksConfig)
		if test.disksConfig != nil {
			// we do not want to compare protobuf messages, only the data inside
			dataExpected, err := json.Marshal(test.disksConfig)
			if err != nil {
				assert.NoError(t, err)
			}
			dataConverted, err := json.Marshal(disksConfig)
			if err != nil {
				assert.NoError(t, err)
			}
			assert.JSONEq(t, string(dataExpected), string(dataConverted))
		}
		parsedDisksLayout, err := device.ParseDiskLayout(disksConfig)
		assert.NoError(t, err)
		assert.Equal(t, test.layout, parsedDisksLayout)
	}
}
