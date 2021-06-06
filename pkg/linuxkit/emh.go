package linuxkit

import (
	"fmt"
	"github.com/packethost/packngo"
	log "github.com/sirupsen/logrus"
)
// EMHClient contains interfaces for communication with EMH
type EMHClient struct {
	devices packngo.DeviceService
	ports packngo.PortService
}

// NewEMHClient creates a new EMH client
func NewEMHClient() (*EMHClient, error) {
	log.Debugf("Connecting to EMH")
	var emhClient *EMHClient
	client, err := packngo.NewClient()
	if err != nil {
		return emhClient, fmt.Errorf(err.Error())
	}
	emhClient = &EMHClient{
		devices: client.Devices,
		ports: client.Ports,
	}
	return emhClient, nil
}

// CreateDevice create a device in EMH
func (emh EMHClient) CreateDevice(projectID, hostname, facility, plan, operatingSystem, ipxeURL string) (*packngo.Device, error) {
	var facilityArgs []string
	if facility != "" {
		facilityArgs = append(facilityArgs, facility)
	}

	request := &packngo.DeviceCreateRequest{
		Hostname: hostname,
		Facility: facilityArgs,
		OS: operatingSystem,
		ProjectID: projectID,
		IPXEScriptURL: ipxeURL,
		Plan: plan,
		BillingCycle: "hourly",
	}

	device, _, err := emh.devices.Create(request)
	return device, err
}

// GetDevice Getting a device
func (emh EMHClient) GetDevice(deviceID string) (*packngo.Device, error) {
	device,_, err := emh.devices.Get(deviceID, nil)
	return device, err
}

// DeleteDevice deletes a device
func (emh EMHClient) DeleteDevice(deviceID string) error {
	_, err := emh.devices.Delete(deviceID, true)
	return err
}

// GetDevicePortByName Getting a device port by name
func (emh EMHClient) GetDevicePortByName(deviceID, portName string) (*packngo.Port, error) {
	device,_, err := emh.devices.Get(deviceID, nil)
	if err != nil {
		return nil, err
	}
	port, err := device.GetPortByName(portName)
	return port, err
}

// DisbondPort disables bonding for one port
func (emh EMHClient) DisbondPort(portID string) error {
	_, _, err := emh.ports.Disbond(portID, false)
	return err
}

// AssignPort adds a VLAN to a port
func (emh EMHClient) AssignPort(portID, vlanID string) error {
	_, _, err := emh.ports.Assign(portID, vlanID)
	return err
}

// AssignNativePort assigns a virtual network to the port as a "native VLAN"
func (emh EMHClient) AssignNativePort(portID, vlanID string) error {
	_, _, err := emh.ports.AssignNative(portID, vlanID)
	return err
}