package expect

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"

	"github.com/lf-edge/eden/pkg/controller"
	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/device"
	"github.com/lf-edge/eden/pkg/utils"
	"github.com/lf-edge/eve-api/go/config"
	"github.com/lf-edge/eve-api/go/evecommon"
	log "github.com/sirupsen/logrus"
)

func (exp *AppExpectation) applyUserData(appInstanceConfig *config.AppInstanceConfig) {
	if exp.metadata == "" {
		return
	}
	userData := base64.StdEncoding.EncodeToString([]byte(exp.metadata))
	encBlock := &evecommon.EncryptionBlock{}
	encBlock.ProtectedUserData = userData
	cipherBlock, err := exp.prepareCipherData(encBlock)
	if err != nil {
		log.Fatal(err)
	}
	if cipherBlock != nil {
		appInstanceConfig.CipherData = cipherBlock
	} else {
		appInstanceConfig.UserData = userData
	}
}

func (exp *AppExpectation) applyDatastoreCipher(datastoreConfig *config.DatastoreConfig) {
	if datastoreConfig.Password == "" && datastoreConfig.ApiKey == "" {
		return
	}
	encBlock := &evecommon.EncryptionBlock{}
	encBlock.DsAPIKey = datastoreConfig.ApiKey
	encBlock.DsPassword = datastoreConfig.Password
	cipherBlock, err := exp.prepareCipherData(encBlock)
	if err != nil {
		log.Fatal(err)
	}
	if cipherBlock != nil {
		datastoreConfig.CipherData = cipherBlock
		datastoreConfig.ApiKey = ""
		datastoreConfig.Password = ""
	}
}

func (exp *AppExpectation) prepareCipherData(encBlock *evecommon.EncryptionBlock) (*evecommon.CipherBlock, error) {
	cipherBlock, err := EncryptForDevice(exp.ctrl, exp.device, encBlock, exp.useEncryptCert)
	if err != nil {
		// historical behavior: fall back to plaintext if encryption is not possible
		log.Errorf("cannot encrypt for device, will use plaintext. error: %s", err)
		return nil, nil
	}
	return cipherBlock, nil
}

// EncryptForDevice encrypts an EncryptionBlock against the device's ECDH
// certificate, registering the resulting CipherContext on dev so it is included
// in the pushed config. Unlike the app-expectation path it returns a hard error
// (no plaintext fallback) so callers that require encryption fail loudly.
// useEncryptCert selects the controller's encrypt.pem (CONTROLLER_ECDH_EXCHANGE)
// vs the signing cert for the ECDH derivation.
func EncryptForDevice(ctrl controller.Cloud, dev *device.Ctx, encBlock *evecommon.EncryptionBlock, useEncryptCert bool) (*evecommon.CipherBlock, error) {
	devCert, err := ctrl.GetECDHCert(dev.GetID())
	if err != nil {
		return nil, fmt.Errorf("GetECDHCert: %w", err)
	}
	ctrlCert, ctrlPrivKey, err := loadControllerCryptoMaterial(ctrl, useEncryptCert)
	if err != nil {
		return nil, fmt.Errorf("loadControllerCryptoMaterial: %w", err)
	}
	cryptoConfig, err := utils.GetCommonCryptoConfig(devCert, ctrlCert, ctrlPrivKey)
	if err != nil {
		return nil, fmt.Errorf("GetCommonCryptoConfig: %w", err)
	}
	cipherCtx, err := utils.CreateCipherCtx(cryptoConfig)
	if err != nil {
		return nil, fmt.Errorf("CreateCipherCtx: %w", err)
	}
	cipherCtx = utils.AddCipherCtxToDev(dev, cipherCtx)
	return utils.CryptoConfigWrapper(encBlock, cryptoConfig, cipherCtx)
}

// loadControllerCryptoMaterial returns the controller cert + matching private
// key to use for ECDH derivation. When useEncryptCert is true, encrypt.pem +
// encrypt-key.pem are used so the resulting cipher context references the
// controller's CONTROLLER_ECDH_EXCHANGE cert; otherwise signing.pem +
// signing-key.pem are used (historical default).
func loadControllerCryptoMaterial(ctrl controller.Cloud, useEncryptCert bool) ([]byte, []byte, error) {
	edenHome, err := utils.DefaultEdenDir()
	if err != nil {
		return nil, nil, fmt.Errorf("DefaultEdenDir: %w", err)
	}
	var (
		certBytes []byte
		keyName   string
	)
	if useEncryptCert {
		certBytes, err = ctrl.EncryptCertGet()
		if err != nil {
			return nil, nil, fmt.Errorf("EncryptCertGet: %w", err)
		}
		keyName = "encrypt-key.pem"
	} else {
		certBytes, err = ctrl.SigningCertGet()
		if err != nil {
			return nil, nil, fmt.Errorf("SigningCertGet: %w", err)
		}
		keyName = "signing-key.pem"
	}
	keyPath := filepath.Join(edenHome, defaults.DefaultCertsDist, keyName)
	keyBytes, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot read %s: %w", keyPath, err)
	}
	return certBytes, keyBytes, nil
}
