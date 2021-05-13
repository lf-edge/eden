package expect

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
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
	attestData, err := exp.ctrl.CertsGet(exp.device.GetID())
	if err != nil {
		log.Errorf("cannot get attestation certificates from cloud for %s will use plaintext", exp.device.GetID())
		appInstanceConfig.UserData = userData
	} else {
		edenHome, err := utils.DefaultEdenDir()
		if err != nil {
			log.Fatalf("DefaultEdenDir: %s", err)
		}
		req := &types.Zcerts{}
		if err := json.Unmarshal([]byte(attestData), req); err != nil {
			log.Fatalf("cannot unmarshal attest: %v", err)
		}
		var cert []byte
		for _, c := range req.Certs {
			if c.Type == certs.ZCertType_CERT_TYPE_DEVICE_ECDH_EXCHANGE {
				cert = c.Cert
			}
		}
		if len(cert) == 0 {
			log.Fatalf("no DEVICE_ECDH_EXCHANGE certificate")
		}
		globalCertsDir := filepath.Join(edenHome, defaults.DefaultCertsDist)
		cryptoConfig, err := utils.GetCommonCryptoConfig(cert, filepath.Join(globalCertsDir, "signing.pem"), filepath.Join(globalCertsDir, "signing-key.pem"))
		if err != nil {
			log.Fatalf("GetCommonCryptoConfig: %v", err)
		}
		encBlock := &config.EncryptionBlock{}
		encBlock.ProtectedUserData = userData
		cipherCtx, err := utils.CreateCipherCtx(cryptoConfig)
		if err != nil {
			log.Fatalf("CreateCipherCtx: %v", err)
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
			log.Fatalf("CryptoConfigWrapper: %v", err)
		}
		appInstanceConfig.CipherData = cipherBlock
	}
}
