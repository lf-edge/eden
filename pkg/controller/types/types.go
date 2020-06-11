package types

//DeviceStateFilter for filter device by state
type DeviceStateFilter int

var (
	AllDevicesFilter          DeviceStateFilter = 0 //return all devices
	RegisteredDeviceFilter    DeviceStateFilter = 1 //return registered devices
	NotRegisteredDeviceFilter DeviceStateFilter = 2 //return not registered devices
)
