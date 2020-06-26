package utils

import (
	"crypto/sha256"
	"encoding/hex"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"
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

//TouchFile create empty file
func TouchFile(src string) (err error) {
	if _, err := os.Stat(src); os.IsNotExist(err) {
		file, err := os.Create(src)
		if err != nil {
			return err
		}
		defer file.Close()
	} else {
		currentTime := time.Now().Local()
		err = os.Chtimes(src, currentTime, currentTime)
		if err != nil {
			return err
		}
	}
	return nil
}

func fileNameWithoutExtension(fileName string) string {
	return strings.TrimSuffix(fileName, filepath.Ext(fileName))
}

//ResolveAbsPath use eden.root parameter to resolve path
func ResolveAbsPath(curPath string) string {
	if strings.TrimSpace(curPath) == "" {
		return ""
	}
	if !filepath.IsAbs(curPath) {
		return filepath.Join(viper.GetString("eden.root"), strings.TrimSpace(curPath))
	}
	return curPath
}

//GetFileFollowLinks resolve file by walking through symlinks
func GetFileFollowLinks(filePath string) (string, error) {
	log.Debugf("GetFileFollowLinks %s", filePath)
	fileInfo, err := os.Lstat(filePath)
	if os.IsNotExist(err) {
		return "", err
	}
	if fileInfo.Mode()&os.ModeSymlink != 0 {
		originFile, err := os.Readlink(filepath.Join(filepath.Dir(filePath), fileInfo.Name()))
		if err != nil {
			return "", err
		}
		fileToSearch := originFile
		if _, err := os.Lstat(fileToSearch); os.IsNotExist(err) {
			fileToSearch = filepath.Join(filepath.Dir(filePath), originFile)
		}
		return GetFileFollowLinks(fileToSearch)
	}
	return filepath.Join(filepath.Dir(filePath), fileInfo.Name()), nil
}

//GetFileSize returns file size
func GetFileSize(filePath string) int64 {
	fi, err := os.Stat(filePath)
	if err != nil {
		log.Fatal(err)
	}
	return fi.Size()
}
