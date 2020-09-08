package expect

import (
	"strings"

	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eve/api/go/config"
	log "github.com/sirupsen/logrus"
)

//VolumeType defines type of empty volumes to use
type VolumeType string

//VolumeQcow2 use empty qcow2 image for volumes
var VolumeQcow2 VolumeType = "qcow2"

//VolumeOCI use empty oci image for volumes
var VolumeOCI VolumeType = "oci"

//VolumeTypeByName returns VolumeType by name
func VolumeTypeByName(name string) VolumeType {
	switch name {
	case "qcow2":
		return VolumeQcow2
	case "oci":
		return VolumeOCI
	default:
		log.Fatalf("Not supported volume type %s", name)
	}
	return VolumeQcow2
}

//ExpectationOption is type to use for creation of appExpectation
type ExpectationOption func(expectation *appExpectation)

//WithVnc enables VNC and sets VNC display number
func WithVnc(vncDisplay uint32) ExpectationOption {
	return func(expectation *appExpectation) {
		expectation.vncDisplay = vncDisplay
	}
}

//WithVncPassword sets VNC password
func WithVncPassword(password string) ExpectationOption {
	return func(expectation *appExpectation) {
		expectation.vncPassword = password
	}
}

//WithMetadata sets metadata for created apps
func WithMetadata(metadata string) ExpectationOption {
	return func(expectation *appExpectation) {
		expectation.metadata = strings.Replace(metadata, `\n`, "\n", -1)
	}
}

//WithAppAdapters assigns adapters for created apps
func WithAppAdapters(appadapters []string) ExpectationOption {
	return func(expectation *appExpectation) {
		expectation.appAdapters = appadapters
	}
}


//AddNetInstanceNameAndPortPublish adds NetInstance with defined name and ports mapping for apps in format ["EXTERNAL_PORT:INTERNAL_PORT"]
func AddNetInstanceNameAndPortPublish(netInstanceName string, portPublish []string) ExpectationOption {
	return func(expectation *appExpectation) {
		expectation.netInstances = append(expectation.netInstances, &netInstanceExpectation{
			name:          netInstanceName,
			portsReceived: portPublish,
			ports:         make(map[int]int),
		})
	}
}

//AddNetInstanceAndPortPublish adds NetInstance with defined subnet cidr, networkType and ports mapping for apps in format ["EXTERNAL_PORT:INTERNAL_PORT"]
func AddNetInstanceAndPortPublish(subnetCidr string, networkType string, portPublish []string) ExpectationOption {
	return func(expectation *appExpectation) {
		expectation.netInstances = append(expectation.netInstances, &netInstanceExpectation{
			subnet:        subnetCidr,
			portsReceived: portPublish,
			ports:         make(map[int]int),
			netInstType:   networkType,
		})
	}
}

//WithPortsPublish sets ports mapping for apps in format ["EXTERNAL_PORT:INTERNAL_PORT"]
func WithPortsPublish(portPublish []string) ExpectationOption {
	return func(expectation *appExpectation) {
		if len(expectation.netInstances) == 0 {
			expectation.netInstances = []*netInstanceExpectation{{
				subnet: defaults.DefaultAppSubnet,
				ports:  make(map[int]int),
			}}
		}
		expectation.netInstances[0].portsReceived = portPublish
	}
}

//WithDiskSize set disk size for created app (equals with image size if not defined)
func WithDiskSize(diskSizeBytes int64) ExpectationOption {
	return func(expectation *appExpectation) {
		expectation.diskSize = diskSizeBytes
	}
}

//WithResources sets cpu count and memory for app
func WithResources(cpus uint32, memory uint32) ExpectationOption {
	return func(expectation *appExpectation) {
		expectation.cpu = cpus
		expectation.mem = memory
	}
}

//WithVirtualizationMode sets virtualizationMode for app
func WithVirtualizationMode(virtualizationMode config.VmMode) ExpectationOption {
	return func(expectation *appExpectation) {
		expectation.virtualizationMode = virtualizationMode
	}
}

// WithImageFormat sets app format
func WithImageFormat(format string) ExpectationOption {
	return func(expectation *appExpectation) {
		expectation.imageFormat = format
	}
}

//WithVolumeType sets empty volumes type for app
func WithVolumeType(volumesType VolumeType) ExpectationOption {
	return func(expectation *appExpectation) {
		expectation.volumesType = volumesType
	}
}

//WithAcl sets access for app only to external networks if onlyHost sets
func WithAcl(onlyHost bool) ExpectationOption {
	return func(expectation *appExpectation) {
		expectation.onlyHostAcl = onlyHost
	}
}
