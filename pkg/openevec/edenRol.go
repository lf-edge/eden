package openevec

import (
	"encoding/json"
	"fmt"
	"path"
	"strings"

	"github.com/Insei/rolgo"
	"github.com/lf-edge/eden/pkg/defaults"
)

func CreateRent(rolProjectID, rolRentName, rolModel, rolManufacturer, rolIPXEUrl string, cfg *EdenSetupArgs) error {
	client, err := rolgo.NewClient()
	if err != nil {
		return err
	}
	if rolIPXEUrl == "" {
		configPrefix := cfg.ConfigName
		if cfg.ConfigName == defaults.DefaultContext {
			configPrefix = ""
		}
		rolIPXEUrl = fmt.Sprintf("http://%s:%d/%s/ipxe.efi.cfg", cfg.Adam.CertsEVEIP, cfg.Eden.EServer.Port, path.Join("eserver", configPrefix))
		// log.Debugf("ipxe-url is empty, will use default one: %s", packetIPXEUrl)
	}
	r := &rolgo.DeviceRentCreateRequest{Model: rolModel, Manufacturer: rolManufacturer, Name: rolRentName,
		IpxeUrl: rolIPXEUrl}
	rent, err := client.Rents.Create(rolProjectID, r)
	if err == nil {
		fmt.Println(rent.Id)
	} else {
		return fmt.Errorf("unable to create device rent: %w", err)
	}
	return nil
}

func GetRent(rolProjectID, rolRentID string) error {
	client, err := rolgo.NewClient()
	if err != nil {
		return err
	}
	rent, err := client.Rents.Get(rolProjectID, rolRentID)
	if err != nil {
		return err
	}
	rentJSON, err := json.Marshal(rent)
	if err != nil {
		return err
	}
	fmt.Println(string(rentJSON))
	return nil
}

func CloseRent(rolProjectID, rolRentID string) error {
	client, err := rolgo.NewClient()
	if err != nil {
		return err
	}
	err = client.Rents.Release(rolProjectID, rolRentID)
	if err != nil {
		return err
	}
	return nil
}

func GetRentConsoleOutput(rolProjectID, rolRentID string) (string, error) {
	client, err := rolgo.NewClient()
	if err != nil {
		return "", err
	}
	consoleOutput, err := client.Rents.GetConsoleOutput(rolProjectID, rolRentID)
	if err != nil {
		return "", err
	}

	return strings.Join(consoleOutput, "\n"), nil
}
