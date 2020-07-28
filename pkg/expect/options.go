package expect

import (
	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eve/api/go/config"
	"strings"
)

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

//AddNetInstanceAndPortPublish adds NetInstance with defined subnet cidr and ports mapping for apps in format ["EXTERNAL_PORT:INTERNAL_PORT"]
func AddNetInstanceAndPortPublish(subnetCidr string, portPublish []string) ExpectationOption {
	return func(expectation *appExpectation) {
		expectation.netInstances = append(expectation.netInstances, &netInstanceExpectation{
			subnet:        subnetCidr,
			portsReceived: portPublish,
			ports:         make(map[int]int),
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
