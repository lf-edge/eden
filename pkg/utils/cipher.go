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

	//encrypt paintext secret using symmetric key and initial value.
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
		return nil, fmt.Errorf("error marshalling user data: %v", mErr)
	}
	//Fill CipherBlock.
	cipherBlock, cbErr := createCipherBlock(mEncBlock, cipherCtx.ContextId, cmnCryptoCfg, iv[:16])
	if cbErr != nil {
		return nil, fmt.Errorf("error creating cipher block: %v", cbErr)
	}
	return cipherBlock, nil
}
