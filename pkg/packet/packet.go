package packet

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/packethost/packngo"
)

// Client wrapper to use for interaction with packet
type Client struct {
	client    *packngo.Client
	projectID string
}

// NewPacketClient creates new client
func NewPacketClient(apiTokenPath, projectName string) (*Client, error) {
	f, err := os.Open(apiTokenPath)
	if err != nil {
		return nil, err
	}

	apiToken, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}
	c := &Client{client: packngo.NewClientWithAuth("eden packet lib", strings.TrimSpace(string(apiToken)), nil)}
	projects, _, err := c.client.Projects.List(nil)
	if err != nil {
		return nil, err
	}
	projectID := ""
	for _, el := range projects {
		if el.Name == projectName {
			projectID = el.ID
		}
	}
	if projectID == "" {
		return nil, fmt.Errorf("project not found: %s", projectName)
	}
	project, _, err := c.client.Projects.Get(projectID, nil)
	if err != nil {
		return nil, fmt.Errorf("cannot get project: %s", err)
	}
	c.projectID = project.ID
	return c, nil
}

// CreateInstance creates new instance on packet with ipxe
func (c *Client) CreateInstance(name, zone, machineType, ipxeURL string) error {
	req := &packngo.DeviceCreateRequest{
		Hostname:      name,
		Plan:          machineType,
		ProjectID:     c.projectID,
		Facility:      []string{zone},
		IPXEScriptURL: ipxeURL,
		OS:            "custom_ipxe",
		Description:   "eden test vm",
		BillingCycle:  "hourly",
	}
	_, _, err := c.client.Devices.Create(req)
	return err
}

func (c *Client) getDeviceID(name string) (string, error) {
	devices, _, err := c.client.Devices.List(c.projectID, nil)
	if err != nil {
		return "", fmt.Errorf("cannot list devices: %s", err)
	}
	id := ""
	for _, el := range devices {
		if el.Hostname == name {
			id = el.ID
			break
		}
	}
	if id == "" {
		return "", fmt.Errorf("no device found with name %s", name)
	}
	return id, nil
}

// DeleteInstance removes instance from packet
func (c *Client) DeleteInstance(name string) error {
	id, err := c.getDeviceID(name)
	if err != nil {
		return err
	}
	_, err = c.client.Devices.Delete(id, true)
	return err
}

// GetInstanceNatIP returns IP of packet server
func (c *Client) GetInstanceNatIP(name string) (string, error) {
	id, err := c.getDeviceID(name)
	if err != nil {
		return "", err
	}
	d, _, err := c.client.Devices.Get(id, nil)
	if err != nil {
		return "", err
	}
	for _, el := range d.Network {
		if el.Public && el.AddressFamily == 4 {
			return el.Address, nil
		}
	}
	return "", fmt.Errorf("address not found")
}
