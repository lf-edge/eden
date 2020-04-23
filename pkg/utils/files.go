package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

//SHA256SUM calculates sha256 of file
func SHA256SUM(filePath string) (result string, err error) {
	file, err := os.Open(filePath)
	if err != nil {
		return
	}
	defer file.Close()

	hash := sha256.New()
	if _, err = io.Copy(hash, file); err != nil {
		return
	}

	result = hex.EncodeToString(hash.Sum(nil))
	return
}

//CopyFileNotExists copy file from src to dst with same permission if not exists
func CopyFileNotExists(src string, dst string) (err error) {
	if _, err = os.Lstat(dst); os.IsNotExist(err) {
		if err = CopyFile(src, dst); err != nil {
			return err
		}
	}
	return nil
}

//CopyFile copy file from src to dst with same permission
func CopyFile(src string, dst string) (err error) {
	info, err := os.Lstat(src)
	if err != nil {
		return err
	}
	if _, err = os.Lstat(dst); os.IsNotExist(err) {
		if err = os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
			return err
		}
	}
	srcLink := src
	if info.Mode()&os.ModeSymlink != 0 {
		//follow symlinks
		srcLink, err = os.Readlink(src)
		if err != nil {
			return err
		}
		srcLink = filepath.Join(filepath.Dir(src), filepath.Base(srcLink))
	}
	data, err := ioutil.ReadFile(srcLink)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(dst, data, info.Mode()^os.ModeSymlink)
}

func fileNameWithoutExtension(fileName string) string {
	return strings.TrimSuffix(fileName, filepath.Ext(fileName))
}
