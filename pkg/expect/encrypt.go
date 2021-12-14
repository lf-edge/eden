package expect

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/lf-edge/eden/pkg/controller/types"
	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/utils"
	"github.com/lf-edge/eve/api/go/certs"
	"github.com/lf-edge/eve/api/go/config"
	log "github.com/sirupsen/logrus"
)

func (exp *AppExpectation) applyUserData(appInstanceConfig *config.AppInstanceConfig) {
	if exp.metadata == "" {
		return
	}
	userData := base64.StdEncoding.EncodeToString([]byte(exp.metadata))
	encBlock := &config.EncryptionBlock{}
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
	encBlock := &config.EncryptionBlock{}
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

func (exp *AppExpectation) prepareCipherData(encBlock *config.EncryptionBlock) (*config.CipherBlock, error) {
	attestData, err := exp.ctrl.CertsGet(exp.device.GetID())
	if err != nil {
		log.Errorf("cannot get attestation certificates from cloud for %s will use plaintext", exp.device.GetID())
		return nil, nil
	}
	edenHome, err := utils.DefaultEdenDir()
	if err != nil {
		return nil, fmt.Errorf("DefaultEdenDir: %s", err)
	}
	req := &types.Zcerts{}
	if err := json.Unmarshal([]byte(attestData), req); err != nil {
		return nil, fmt.Errorf("cannot unmarshal attest: %v", err)
	}
	var cert []byte
	for _, c := range req.Certs {
		if c.Type == certs.ZCertType_CERT_TYPE_DEVICE_ECDH_EXCHANGE {
			cert = c.Cert
		}
	}
	if len(cert) == 0 {
		return nil, fmt.Errorf("no DEVICE_ECDH_EXCHANGE certificate")
	}
	globalCertsDir := filepath.Join(edenHome, defaults.DefaultCertsDist)
	cryptoConfig, err := utils.GetCommonCryptoConfig(cert, filepath.Join(globalCertsDir, "signing.pem"), filepath.Join(globalCertsDir, "signing-key.pem"))
	if err != nil {
		return nil, fmt.Errorf("GetCommonCryptoConfig: %v", err)
	}
	cipherCtx, err := utils.CreateCipherCtx(cryptoConfig)
	if err != nil {
		return nil, fmt.Errorf("CreateCipherCtx: %v", err)
	}
	appendCipherCtx := true
	for _, c := range exp.device.GetCipherContexts() {
		// we do not change controller certificates
		if bytes.Equal(c.DeviceCertHash, cipherCtx.DeviceCertHash) {
			cipherCtx = c
			appendCipherCtx = false
		}
	}
	if appendCipherCtx {
		exp.device.SetCipherContexts(append(exp.device.GetCipherContexts(), cipherCtx))
	}
	cipherBlock, err := utils.CryptoConfigWrapper(encBlock, cryptoConfig, cipherCtx)
	if err != nil {
		return nil, fmt.Errorf("CryptoConfigWrapper: %v", err)
	}
	return cipherBlock, nil
}
