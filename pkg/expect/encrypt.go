package expect

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"

	"github.com/lf-edge/eden/pkg/defaults"
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
	// get device certificate from the controller
	devCert, err := exp.ctrl.GetECDHCert(exp.device.GetID())
	if err != nil {
		log.Errorf("cannot get device certificate from cloud. will use plaintext. error: %s", err)
		return nil, nil
	}

	// get signing certificate from the controller
	signCert, err := exp.ctrl.SigningCertGet()
	if err != nil {
		log.Errorf("cannot get cloud's signing certificate. will use plaintext. error: %s", err)
		return nil, nil
	}

	edenHome, err := utils.DefaultEdenDir()
	if err != nil {
		return nil, fmt.Errorf("DefaultEdenDir: %w", err)
	}
	keyPath := filepath.Join(edenHome, defaults.DefaultCertsDist, "signing-key.pem")
	ctrlPrivKey, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("cannot read %s: %w", keyPath, err)
	}

	cryptoConfig, err := utils.GetCommonCryptoConfig(devCert, signCert, ctrlPrivKey)
	if err != nil {
		return nil, fmt.Errorf("GetCommonCryptoConfig: %w", err)
	}
	cipherCtx, err := utils.CreateCipherCtx(cryptoConfig)
	if err != nil {
		return nil, fmt.Errorf("CreateCipherCtx: %w", err)
	}

	// add cipher context to device or return a matching existing one
	cipherCtx = utils.AddCipherCtxToDev(exp.device, cipherCtx)

	return utils.CryptoConfigWrapper(encBlock, cryptoConfig, cipherCtx)
}
