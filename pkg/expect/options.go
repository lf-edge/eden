package expect

import (
	"strings"

	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eve/api/go/config"
)

//VolumeType defines type of empty volumes to use
type VolumeType string

//VolumeQcow2 use empty qcow2 image for volumes
var VolumeQcow2 VolumeType = "qcow2"

//VolumeQcow use empty raw image for volumes
var VolumeQcow VolumeType = "qcow"

//VolumeVMDK use empty raw image for volumes
var VolumeVMDK VolumeType = "vmdk"

//VolumeVHDX use empty raw image for volumes
var VolumeVHDX VolumeType = "vhdx"

//VolumeRaw use empty raw image for volumes
var VolumeRaw VolumeType = "raw"

//VolumeOCI use empty oci image for volumes
var VolumeOCI VolumeType = "oci"

//VolumeNone use no volumes
var VolumeNone VolumeType = "none"

//VolumeTypeByName returns VolumeType by name
func VolumeTypeByName(name string) VolumeType {
	switch name {
	case "qcow2":
		return VolumeQcow2
	case "raw":
		return VolumeRaw
	case "vmdk":
		return VolumeVMDK
	case "vhdx":
		return VolumeVHDX
	case "qcow":
		return VolumeQcow
	case "oci":
		return VolumeOCI
	case "none":
		return VolumeNone
	}
	return VolumeQcow2
}

//ExpectationOption is type to use for creation of AppExpectation
type ExpectationOption func(expectation *AppExpectation)

//WithVnc enables VNC and sets VNC display number
func WithVnc(vncDisplay uint32) ExpectationOption {
	return func(expectation *AppExpectation) {
		expectation.vncDisplay = vncDisplay
	}
}

//WithVncPassword sets VNC password
func WithVncPassword(password string) ExpectationOption {
	return func(expectation *AppExpectation) {
		expectation.vncPassword = password
	}
}

//WithMetadata sets metadata for created apps
func WithMetadata(metadata string) ExpectationOption {
	return func(expectation *AppExpectation) {
		expectation.metadata = strings.Replace(metadata, `\n`, "\n", -1)
	}
}

//WithAppAdapters assigns adapters for created apps
func WithAppAdapters(appadapters []string) ExpectationOption {
	return func(expectation *AppExpectation) {
		expectation.appAdapters = appadapters
	}
}

//AddNetInstanceNameAndPortPublish adds NetInstance with defined name and ports mapping for apps in format ["EXTERNAL_PORT:INTERNAL_PORT"]
func AddNetInstanceNameAndPortPublish(netInstance string, portPublish []string) ExpectationOption {
	mac := ""
	split := strings.Split(netInstance, ":")
	name := split[0]
	if len(split) == 7 {
		mac = strings.Join(split[1:], ":")
	}
	return func(expectation *AppExpectation) {
		expectation.netInstances = append(expectation.netInstances, &NetInstanceExpectation{
			name:          name,
			portsReceived: portPublish,
			ports:         make(map[int]int),
			mac:           mac,
		})
	}
}

//AddNetInstanceAndPortPublish adds NetInstance with defined subnet cidr, networkType,
//netInstanceName and ports mapping for apps in format ["EXTERNAL_PORT:INTERNAL_PORT"]
func AddNetInstanceAndPortPublish(subnetCidr, networkType, netInstanceName string, portPublish []string, uplinkAdapter string) ExpectationOption {
	return func(expectation *AppExpectation) {
		expectation.netInstances = append(expectation.netInstances, &NetInstanceExpectation{
			name:          netInstanceName,
			subnet:        subnetCidr,
			portsReceived: portPublish,
			ports:         make(map[int]int),
			netInstType:   networkType,
			uplinkAdapter: uplinkAdapter,
		})
	}
}

//WithPortsPublish sets ports mapping for apps in format ["EXTERNAL_PORT:INTERNAL_PORT"]
func WithPortsPublish(portPublish []string) ExpectationOption {
	return func(expectation *AppExpectation) {
		if len(expectation.netInstances) == 0 {
			expectation.netInstances = []*NetInstanceExpectation{{
				subnet: defaults.DefaultAppSubnet,
				ports:  make(map[int]int),
			}}
		}
		expectation.netInstances[0].portsReceived = portPublish
	}
}

//WithStaticDNSEntries extends network configuration with static DNS entries
//in format ["HOSTNAME:IP_ADDRESS,IP_ADDRESS,..."]
func WithStaticDNSEntries(networkName string, dnsEntries []string) ExpectationOption {
	return func(expectation *AppExpectation) {
		for _, netInstance := range expectation.netInstances {
			if netInstance.name != networkName {
				continue
			}
			netInstance.staticDNSEntries = make(map[string][]string)
			for _, entry := range dnsEntries {
				mapping := strings.SplitN(entry, ":", 2)
				if len(mapping) != 2 {
					continue
				}
				hostname := mapping[0]
				ips := strings.Split(mapping[1], ",")
				netInstance.staticDNSEntries[hostname] = ips
			}
		}
	}
}

//WithDiskSize set disk size for created app (equals with image size if not defined)
func WithDiskSize(diskSizeBytes int64) ExpectationOption {
	return func(expectation *AppExpectation) {
		expectation.diskSize = diskSizeBytes
	}
}

//WithVolumeSize set volume size for created app
func WithVolumeSize(volumeSizeBytes int64) ExpectationOption {
	return func(expectation *AppExpectation) {
		expectation.volumeSize = volumeSizeBytes
	}
}

//WithResources sets cpu count and memory for app
func WithResources(cpus uint32, memory uint32) ExpectationOption {
	return func(expectation *AppExpectation) {
		expectation.cpu = cpus
		expectation.mem = memory
	}
}

//WithVirtualizationMode sets virtualizationMode for app
func WithVirtualizationMode(virtualizationMode config.VmMode) ExpectationOption {
	return func(expectation *AppExpectation) {
		expectation.virtualizationMode = virtualizationMode
	}
}

// WithImageFormat sets app format
func WithImageFormat(format string) ExpectationOption {
	return func(expectation *AppExpectation) {
		expectation.imageFormat = format
	}
}

//WithVolumeType sets empty volumes type for app
func WithVolumeType(volumesType VolumeType) ExpectationOption {
	return func(expectation *AppExpectation) {
		expectation.volumesType = volumesType
	}
}

//WithACL sets access only for defined hosts
func WithACL(acl ACLs) ExpectationOption {
	return func(expectation *AppExpectation) {
		expectation.acl = acl
	}
}

//WithVLANs sets access VLAN IDs for application interfaces
func WithVLANs(vlans map[string]int) ExpectationOption {
	return func(expectation *AppExpectation) {
		expectation.vlans = vlans
	}
}

//WithRegistry sets registry to use (remote/local)
func WithRegistry(registry string) ExpectationOption {
	return func(expectation *AppExpectation) {
		expectation.registry = registry
	}
}

//WithOldApp sets old app name to get info from
func WithOldApp(appName string) ExpectationOption {
	return func(expectation *AppExpectation) {
		expectation.oldAppName = appName
	}
}

//WithHTTPDirectLoad use eserver only for SHA calculation
func WithHTTPDirectLoad(direct bool) ExpectationOption {
	return func(expectation *AppExpectation) {
		expectation.httpDirectLoad = direct
	}
}

//WithSFTPLoad force eserver to serve image via sftp
func WithSFTPLoad(sftp bool) ExpectationOption {
	return func(expectation *AppExpectation) {
		expectation.sftpLoad = sftp
	}
}

//WithAdditionalDisks adds disks to application
func WithAdditionalDisks(disks []string) ExpectationOption {
	return func(expectation *AppExpectation) {
		expectation.disks = disks
	}
}

//WithOpenStackMetadata use openstackMetadata for VM
func WithOpenStackMetadata(openStackMetadata bool) ExpectationOption {
	return func(expectation *AppExpectation) {
		expectation.openStackMetadata = openStackMetadata
	}
}

//WithProfiles set profileList for appInstance
func WithProfiles(profiles []string) ExpectationOption {
	return func(expectation *AppExpectation) {
		expectation.profiles = profiles
	}
}
