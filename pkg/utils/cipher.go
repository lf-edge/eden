package utils

import (
	"fmt"

	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"

	"github.com/lf-edge/eve-api/go/evecommon"
	"google.golang.org/protobuf/proto"
)

// internal encryption method
func aesEncrypt(iv, symmetricKey, plaintext []byte) ([]byte, error) {
	ciphertext := make([]byte, len(plaintext))
	aesBlockEncrypter, err := aes.NewCipher(symmetricKey)
	if err != nil {
		return nil, err
	}
	aesEncrypter := cipher.NewCFBEncrypter(aesBlockEncrypter, iv)
	aesEncrypter.XORKeyStream(ciphertext, plaintext)
	return ciphertext, nil
}

// create cipher block
func createCipherBlock(plainText []byte, cipherCtxID string, cmnCryptoCfg *CommonCryptoConfig, iv []byte) (*evecommon.CipherBlock, error) {
	if cmnCryptoCfg.DevCertHash == nil {
		return nil, fmt.Errorf("empty device certificate in create cipher block method")
	}
	cipherBlock := &evecommon.CipherBlock{}
	cipherBlock.CipherContextId = cipherCtxID
	shaOfPlainTextSecret := sha256.Sum256(plainText)
	cipherBlock.ClearTextSha256 = shaOfPlainTextSecret[:]
	cipherBlock.InitialValue = iv

	// encrypt paintext secret using symmetric key and initial value.
	cipherText, ecErr := aesEncrypt(iv, cmnCryptoCfg.SymmetricKey, plainText)
	if ecErr != nil {
		return nil, ecErr
	}
	cipherBlock.CipherData = cipherText

	return cipherBlock, nil
}

// CryptoConfigWrapper create cipherCtx and encrypt secrets for all the objects.
func CryptoConfigWrapper(encBlock *evecommon.EncryptionBlock, cmnCryptoCfg *CommonCryptoConfig, cipherCtx *evecommon.CipherContext) (*evecommon.CipherBlock, error) {
	// Prepare initial value by appending device cert hash and controller cert hash
	// and calculate sha of that.
	var concatIV []byte
	concatIV = append(concatIV, cipherCtx.ControllerCertHash[:8]...)
	concatIV = append(concatIV, cipherCtx.DeviceCertHash[:8]...)
	iv := sha256.Sum256(concatIV)

	mEncBlock, mErr := proto.Marshal(encBlock)
	if mErr != nil {
		return nil, fmt.Errorf("error marshalling user data: %w", mErr)
	}
	// Fill CipherBlock.
	cipherBlock, cbErr := createCipherBlock(mEncBlock, cipherCtx.ContextId, cmnCryptoCfg, iv[:16])
	if cbErr != nil {
		return nil, fmt.Errorf("error creating cipher block: %w", cbErr)
	}
	return cipherBlock, nil
}

// internal decryption method
func aesDecrypt(iv, symmetricKey, ciphertext []byte) ([]byte, error) {
	plaintext := make([]byte, len(ciphertext))
	aesBlockDecrypter, err := aes.NewCipher(symmetricKey)
	if err != nil {
		return nil, err
	}
	aesDecrypter := cipher.NewCFBDecrypter(aesBlockDecrypter, iv)
	aesDecrypter.XORKeyStream(plaintext, ciphertext)
	return plaintext, nil
}

// reverse the process of creating a cipher block
func decryptCipherBlock(cipherBlock *evecommon.CipherBlock, cmnCryptoCfg *CommonCryptoConfig) ([]byte, error) {
	if cipherBlock == nil {
		return nil, fmt.Errorf("nil cipher block in decrypt cipher block method")
	}

	// decrypt ciphertext using symmetric key and initial value
	decryptedText, decErr := aesDecrypt(cipherBlock.InitialValue, cmnCryptoCfg.SymmetricKey, cipherBlock.CipherData)
	if decErr != nil {
		return nil, decErr
	}

	// verify sha256 checksum of decrypted text
	shaOfDecryptedText := sha256.Sum256(decryptedText)
	if !CompareSlices(shaOfDecryptedText[:], cipherBlock.ClearTextSha256) {
		return nil, fmt.Errorf("decrypted text does not match original sha256 checksum")
	}

	return decryptedText, nil
}

// CryptoConfigUnwrapper reverses the process of CryptoConfigWrapper
func CryptoConfigUnwrapper(cipherBlock *evecommon.CipherBlock, cmnCryptoCfg *CommonCryptoConfig) (*evecommon.EncryptionBlock, error) {
	decryptedBytes, err := decryptCipherBlock(cipherBlock, cmnCryptoCfg)
	if err != nil {
		return nil, fmt.Errorf("error decrypting cipher block: %w", err)
	}

	var encBlock evecommon.EncryptionBlock
	if err := proto.Unmarshal(decryptedBytes, &encBlock); err != nil {
		return nil, fmt.Errorf("error unmarshalling decrypted bytes: %w", err)
	}

	return &encBlock, nil
}

// CipherDataHolder is an interface for objects that have CipherData field.
type CipherDataHolder interface {
	GetCipherData() *evecommon.CipherBlock
}

// ReencryptConfigData re-encrypts config data with new crypto config.
func ReencryptConfigData(holder CipherDataHolder, oldCryptoConfig, newCryptoConfig *CommonCryptoConfig, cipherCtx *evecommon.CipherContext) error {
	if cipherData := holder.GetCipherData(); cipherData != nil {
		encBlock, err := CryptoConfigUnwrapper(cipherData, oldCryptoConfig)
		if err != nil {
			return fmt.Errorf("CryptoConfigUnwrapper error: %w", err)
		}

		newCipherData, err := CryptoConfigWrapper(encBlock, newCryptoConfig, cipherCtx)
		if err != nil {
			return fmt.Errorf("CryptoConfigWrapper error: %w", err)
		}

		// copy each field separately to avoid copying the lock in MessageState
		cipherData.CipherContextId = newCipherData.CipherContextId
		cipherData.InitialValue = newCipherData.InitialValue
		cipherData.CipherData = newCipherData.CipherData
		cipherData.ClearTextSha256 = newCipherData.ClearTextSha256
	}
	return nil
}
