package utils

import (
	"crypto/sha256"
	"fmt"
	"os"
	"strings"

	"github.com/google/uuid"
	"github.com/lf-edge/eve/api/go/config"
	"github.com/lf-edge/eve/api/go/evecommon"
)

// CommonCryptoConfig stores information about certificates
type CommonCryptoConfig struct {
	ControllerEncCertHash []byte
	DevCertHash           []byte
	SymmetricKey          []byte
}

// GetCommonCryptoConfig calculate common crypto config
// and keep it in a structure.
// Common config are:
// 1. Calculate sha of controller cert.
// 2. Calculate sha of device cert.
// 3. Calculate symmetric key.
func GetCommonCryptoConfig(devCert []byte, controllerCert, controllerKey string) (*CommonCryptoConfig, error) {
	ctrlEncCert, rErr := os.ReadFile(controllerCert)
	if rErr != nil {
		return nil, rErr
	}
	//first trim space from controller cert before calculating hash.
	strCtrlEncCert := string(ctrlEncCert)
	controllerEncCertSha := sha256.Sum256([]byte(strings.TrimSpace(strCtrlEncCert)))

	//calculate sha256 of devCert.
	devCertSha := sha256.Sum256(devCert)

	//read controller encryption priv key and
	//use it for computing symmetric key.
	ctrlPrivKey, rErr := os.ReadFile(controllerKey)
	if rErr != nil {
		return nil, rErr
	}
	//calculate symmetric key.
	symmetricKey, syErr := calculateSymmetricKeyForEcdhAES(devCert, ctrlPrivKey)
	if syErr != nil {
		return nil, syErr
	}

	ccc := &CommonCryptoConfig{}
	ccc.ControllerEncCertHash = controllerEncCertSha[:]
	ccc.DevCertHash = devCertSha[:]
	ccc.SymmetricKey = symmetricKey
	return ccc, nil
}

// CreateCipherCtx for edge dev config.
func CreateCipherCtx(cmnCryptoCfg *CommonCryptoConfig) (*config.CipherContext, error) {
	if cmnCryptoCfg.DevCertHash == nil {
		return nil, fmt.Errorf("Empty device certificate in create cipher context method")
	}

	cipherCtx := &config.CipherContext{}

	//prepare ctx using controller and device cert hash.
	//append device cert has and controller cert hash.
	var uid uuid.UUID
	appendedHash := append(cmnCryptoCfg.ControllerEncCertHash[:16], cmnCryptoCfg.DevCertHash[:16]...)
	ctxID := uuid.NewSHA1(uid, appendedHash)
	cipherCtx.ContextId = ctxID.String()

	cipherCtx.HashScheme = evecommon.HashAlgorithm_HASH_ALGORITHM_SHA256_16BYTES
	cipherCtx.KeyExchangeScheme = config.KeyExchangeScheme_KEA_ECDH
	cipherCtx.EncryptionScheme = config.EncryptionScheme_SA_AES_256_CFB

	cipherCtx.DeviceCertHash = cmnCryptoCfg.DevCertHash[:16]
	cipherCtx.ControllerCertHash = cmnCryptoCfg.ControllerEncCertHash[:16]

	return cipherCtx, nil
}
