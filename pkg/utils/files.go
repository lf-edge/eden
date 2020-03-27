package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
)

//SHA256SUM calculates sha256 of file
func SHA256SUM(filePath string) (result string, err error) {
	file, err := os.Open(filePath)
	if err != nil {
		return
	}
	defer file.Close()

	hash := sha256.New()
	_, err = io.Copy(hash, file)
	if err != nil {
		return
	}

	result = hex.EncodeToString(hash.Sum(nil))
	return
}
