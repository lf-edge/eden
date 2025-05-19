package openevec

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/lf-edge/eden/pkg/controller"
	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/device"
	"github.com/lf-edge/eden/pkg/eden"
	"github.com/lf-edge/eden/pkg/utils"
	log "github.com/sirupsen/logrus"
)

// AdamStart starts the OpenEVEC controller.
func (openEVEC *OpenEVEC) AdamStart() error {
	cfg := openEVEC.cfg
	command, err := os.Executable()
	if err != nil {
		return fmt.Errorf("cannot obtain executable path: %w", err)
	}
	log.Infof("Executable path: %s", command)
	if !cfg.Adam.Remote.Redis {
		cfg.Adam.Redis.RemoteURL = ""
	}
	if err := eden.StartAdam(cfg.Adam.Port, cfg.Adam.Dist, cfg.Adam.Force, cfg.Adam.Tag,
		cfg.Adam.Redis.RemoteURL, cfg.Adam.APIv1, cfg.Eden.EnableIPv6, cfg.Eden.IPv6Subnet); err != nil {
		log.Errorf("cannot start adam: %s", err.Error())
	} else {
		log.Infof("Adam is running and accessible on port %d", cfg.Adam.Port)
	}
	return nil
}

// ChangeSigningCert uploads the provided signing certificate to the OpenEVEC controller.
func (openEVEC *OpenEVEC) ChangeSigningCert(newSignCert []byte) error {
	changer := &adamChanger{}
	ctrl, dev, err := changer.getControllerAndDevFromConfig(openEVEC.cfg)
	if err != nil {
		return fmt.Errorf("getControllerAndDevFromConfig: %w", err)
	}

	// we need to re-encrypt existing configs with the new certificate because EVE has support only for one server signing certificate
	err = reencryptConfigs(ctrl, dev, newSignCert)
	if err != nil {
		return fmt.Errorf("failed to reencrypt existing configs: %w", err)
	}

	if err = changer.setControllerAndDev(ctrl, dev); err != nil {
		return fmt.Errorf("setControllerAndDev: %w", err)
	}

	edenHome, err := utils.DefaultEdenDir()
	if err != nil {
		return err
	}
	globalCertsDir := filepath.Join(edenHome, defaults.DefaultCertsDist)
	signingCertPath := filepath.Join(globalCertsDir, "signing.pem")

	if err = os.WriteFile(signingCertPath, newSignCert, 0644); err != nil {
		return fmt.Errorf("cannot write signing cert to %s: %w", signingCertPath, err)
	}

	log.Infof("Signing cert changed successfully")
	return nil
}

func reencryptConfigs(ctrl controller.Cloud, dev *device.Ctx, newSignCert []byte) error {
	// get device certificate from the controller
	devCert, err := ctrl.GetECDHCert(dev.GetID())
	if err != nil {
		return fmt.Errorf("cannot get device certificate from cloud: %w", err)
	}

	// get signing certificate from the controller
	oldSignCert, err := ctrl.SigningCertGet()
	if err != nil {
		log.Error("cannot get cloud's signing certificate. will use plaintext")
		return nil
	}

	edenHome, err := utils.DefaultEdenDir()
	if err != nil {
		return fmt.Errorf("DefaultEdenDir: %w", err)
	}
	keyPath := filepath.Join(edenHome, defaults.DefaultCertsDist, "signing-key.pem")
	ctrlPrivKey, err := os.ReadFile(keyPath)
	if err != nil {
		return fmt.Errorf("cannot read %s: %w", keyPath, err)
	}

	oldCryptoConfig, err := utils.GetCommonCryptoConfig(devCert, oldSignCert, ctrlPrivKey)
	if err != nil {
		return fmt.Errorf("GetCommonCryptoConfig: %w", err)
	}

	newCryptoConfig, err := utils.GetCommonCryptoConfig(devCert, newSignCert, ctrlPrivKey)
	if err != nil {
		return fmt.Errorf("GetCommonCryptoConfig: %w", err)
	}

	cipherCtx, err := utils.CreateCipherCtx(newCryptoConfig)
	if err != nil {
		return fmt.Errorf("CreateCipherCtx: %w", err)
	}
	// add cipher context to device or return a matching existing one
	cipherCtx = utils.AddCipherCtxToDev(dev, cipherCtx)

	// re-encrypt all app configs with the new signing certificate
	appConfigs := ctrl.ListApplicationInstanceConfig()
	for _, config := range appConfigs {
		if err = utils.ReencryptConfigData(config, oldCryptoConfig, newCryptoConfig, cipherCtx); err != nil {
			return fmt.Errorf("reencryptConfigData: %w", err)
		}
	}

	// re-encrypt all datastore configs with the new signing certificate
	dsConfigs := ctrl.ListDataStore()
	for _, config := range dsConfigs {
		if err = utils.ReencryptConfigData(config, oldCryptoConfig, newCryptoConfig, cipherCtx); err != nil {
			return fmt.Errorf("reencryptConfigData: %w", err)
		}
	}

	// re-encrypt all wireless configs with the new signing certificate
	for _, networkConfigID := range dev.GetNetworks() {
		networkConfig, err := ctrl.GetNetworkConfig(networkConfigID)
		if err != nil {
			return fmt.Errorf("GetNetworkConfig: %w", err)
		}
		if networkConfig != nil && networkConfig.Wireless != nil {
			for _, config := range networkConfig.Wireless.CellularCfg {
				for _, ap := range config.AccessPoints {
					if err = utils.ReencryptConfigData(ap, oldCryptoConfig, newCryptoConfig, cipherCtx); err != nil {
						return fmt.Errorf("reencryptConfigData: %w", err)
					}
				}
			}
			for _, config := range networkConfig.Wireless.WifiCfg {
				if err = utils.ReencryptConfigData(config, oldCryptoConfig, newCryptoConfig, cipherCtx); err != nil {
					return fmt.Errorf("reencryptConfigData: %w", err)
				}
			}
		}
	}

	return nil
}
