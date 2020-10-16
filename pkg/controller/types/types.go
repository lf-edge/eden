package types

import (
	"fmt"
	uuid "github.com/satori/go.uuid"
)

//DeviceStateFilter for filter device by state
type DeviceStateFilter int

var (
	// AllDevicesFilter returns all devices
	AllDevicesFilter DeviceStateFilter
	// RegisteredDeviceFilter returns registered devices
	RegisteredDeviceFilter DeviceStateFilter = 1
	// NotRegisteredDeviceFilter returns not registered devices
	NotRegisteredDeviceFilter DeviceStateFilter = 2
)

//PrintResult for representation of printing info/log/metric
// it contains print path string as a key of map
// []string as result of resolving of path string
type PrintResult map[string][]string

func (pr *PrintResult) getMap() map[string][]string {
	return *pr
}

//Print of PrintResult perform output of element of info/log/metric
// if one path string return it
// if multiple path string return them with : as delimiter between key and value and \t as delimiter between path strings
// if one element for path string result, return it as plain string
// if multiple elements, return them as array
func (pr *PrintResult) Print() {
	switch len(*pr) {
	case 0:
		return
	case 1:
		for _, el := range pr.getMap() {
			if len(el) == 1 {
				fmt.Println(el[0])
			} else {
				fmt.Println(el)
			}
			return
		}
	default:
		for k, el := range pr.getMap() {
			if len(el) == 1 {
				fmt.Printf("%s:%s\t", k, el[0])
			} else {
				fmt.Printf("%s:%s\t", k, el)
			}
		}
		fmt.Println()
	}
}

type getDir = func(devUUID uuid.UUID) (dir string)

type getDirApps = func(devUUID uuid.UUID, appUUID uuid.UUID) (dir string)

// DirGetters provides information about directories to obtain objects from for loaders
type DirGetters struct {
	LogsGetter    getDir
	InfoGetter    getDir
	MetricsGetter getDir
	RequestGetter getDir
	AppsGetter    getDirApps
}

type getStream = func(devUUID uuid.UUID) (stream string)

type getStreamApps = func(devUUID uuid.UUID, appUUID uuid.UUID) (stream string)

// StreamGetters provides information about redis streams to obtain objects from for loaders
type StreamGetters struct {
	StreamLogs    getStream
	StreamInfo    getStream
	StreamMetrics getStream
	StreamRequest getStream
	StreamApps    getStreamApps
}

type getURL = func(devUUID uuid.UUID) (url string)

type getURLApps = func(devUUID uuid.UUID, appUUID uuid.UUID) (url string)

// URLGetters provides information about urls to obtain objects from for loaders
type URLGetters struct {
	URLLogs    getURL
	URLInfo    getURL
	URLMetrics getURL
	URLRequest getURL
	URLApps    getURLApps
}

//LoaderObjectType for determinate object for loaders
type LoaderObjectType int

//LogsType for observe logs
var LogsType LoaderObjectType = 1

//InfoType for observe info
var InfoType LoaderObjectType = 2

//MetricsType for observe metrics
var MetricsType LoaderObjectType = 3

//RequestType for observe requests
var RequestType LoaderObjectType = 4

//AppsType for observe logs of apps
var AppsType LoaderObjectType = 5
