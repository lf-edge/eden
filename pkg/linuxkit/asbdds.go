package linuxkit

import (
	"github.com/go-resty/resty/v2"
)

const serverIP = "10.10.98.2"
const serverPort = "5000"

// ASBDDSClient contains state required for communication with ASBDDS
type ASBDDSClient struct {
	serverIP string
	serverPort string
	APIBaseURL string
	rest resty.Client
}

// NewASBDDSClient creates a new ASBDDS client
func NewASBDDSClient() (*ASBDDSClient, error) {
	var client = &ASBDDSClient{
		serverIP: serverIP,
		serverPort: serverPort,
		rest: *resty.New(),
		APIBaseURL: "http://" + serverIP + ":" + serverPort + "/",
	}
	return client, nil
}

// CreateDevice create a device in asbdds
func (a ASBDDSClient) CreateDevice(model, ipxeURL string) (string, error){
	resp, err := a.rest.R().
		SetQueryParams(map[string]string{
			"model": model,
			"ipxe_url": ipxeURL,
		}).
		SetHeader("Accept", "application/json").
		Put(a.APIBaseURL + "device")

	return resp.String(), err
}

// DeleteDevice delete a device in asbdds
func (a ASBDDSClient) DeleteDevice(deviceUUID string) (string, error){
	resp, err := a.rest.R().
		SetHeader("Accept", "application/json").
		Delete(a.APIBaseURL + "device/" + deviceUUID)

	return resp.String(), err
}
