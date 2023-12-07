package utils

import (
	"crypto/sha256"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/lf-edge/eden/pkg/device"
	"github.com/lf-edge/eve-api/go/evecommon"
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
func GetCommonCryptoConfig(devCert, signCert, controllerKey []byte) (*CommonCryptoConfig, error) {
	// first trim space from controller cert before calculating hash.
	strCtrlEncCert := string(signCert)
	controllerEncCertSha := sha256.Sum256([]byte(strings.TrimSpace(strCtrlEncCert)))

	// calculate sha256 of devCert.
	devCertSha := sha256.Sum256(devCert)

	// calculate symmetric key.
	symmetricKey, syErr := calculateSymmetricKeyForEcdhAES(devCert, controllerKey)
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
func CreateCipherCtx(cmnCryptoCfg *CommonCryptoConfig) (*evecommon.CipherContext, error) {
	if cmnCryptoCfg.DevCertHash == nil {
		return nil, fmt.Errorf("empty device certificate in create cipher context method")
	}

	cipherCtx := &evecommon.CipherContext{}

	// prepare ctx using controller and device cert hash.
	// append device cert has and controller cert hash.
	var uid uuid.UUID
	appendedHash := append(cmnCryptoCfg.ControllerEncCertHash[:16], cmnCryptoCfg.DevCertHash[:16]...)
	ctxID := uuid.NewSHA1(uid, appendedHash)
	cipherCtx.ContextId = ctxID.String()

	cipherCtx.HashScheme = evecommon.HashAlgorithm_HASH_ALGORITHM_SHA256_16BYTES
	cipherCtx.KeyExchangeScheme = evecommon.KeyExchangeScheme_KEA_ECDH
	cipherCtx.EncryptionScheme = evecommon.EncryptionScheme_SA_AES_256_CFB

	cipherCtx.DeviceCertHash = cmnCryptoCfg.DevCertHash[:16]
	cipherCtx.ControllerCertHash = cmnCryptoCfg.ControllerEncCertHash[:16]

	return cipherCtx, nil
}

// AddCipherCtxToDev add cipher context to device, unless it already exists.
// It returns the existing or the added cipher context.
func AddCipherCtxToDev(dev *device.Ctx, cipherCtx *evecommon.CipherContext) *evecommon.CipherContext {
	// check if we already have cipherCtx with the same certificates
	for _, c := range dev.GetCipherContexts() {
		sameCipherCtx := CompareSlices(c.DeviceCertHash, cipherCtx.DeviceCertHash) &&
			CompareSlices(c.ControllerCertHash, cipherCtx.ControllerCertHash)
		if sameCipherCtx {
			return c
		}
	}

	dev.SetCipherContexts(append(dev.GetCipherContexts(), cipherCtx))
	return cipherCtx
}
