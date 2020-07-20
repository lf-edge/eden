package types

import (
	"fmt"
)

//DeviceStateFilter for filter device by state
type DeviceStateFilter int

var (
	AllDevicesFilter          DeviceStateFilter = 0 //return all devices
	RegisteredDeviceFilter    DeviceStateFilter = 1 //return registered devices
	NotRegisteredDeviceFilter DeviceStateFilter = 2 //return not registered devices
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
