package manager

import (
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"path/filepath"

	"github.com/lf-edge/eden/eserver/api"
)

// EServerManager for process files
type EServerManager struct {
	Dir string
}

// Init directories for EServerManager
func (mgr *EServerManager) Init() {
	if _, err := os.Stat(mgr.Dir); err != nil {
		if err = os.MkdirAll(mgr.Dir, 0755); err != nil {
			log.Fatal(err)
		}
	}
}

// ListFileNames list downloaded files
func (mgr *EServerManager) ListFileNames() (result []string) {
	files, err := os.ReadDir(mgr.Dir)
	if err != nil {
		log.Fatal(err)
	}
	for _, f := range files {
		result = append(result, f.Name())
	}
	return
}

// getFileSize returns file size
func getFileSize(filePath string) int64 {
	fi, err := os.Stat(filePath)
	if err != nil {
		log.Fatal(err)
	}
	return fi.Size()
}

// downloadFile downloads a url to a local file.
func downloadFile(filePath, url string) error {
	out, err := os.Create(filePath + ".tmp")
	if err != nil {
		return err
	}
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	hash := sha256.New()
	_, err = io.Copy(hash, io.TeeReader(resp.Body, out))
	if err != nil {
		return err
	}
	if err = os.WriteFile(fmt.Sprintf("%s.sha256", filePath), []byte(hex.EncodeToString(hash.Sum(nil))), 0666); err != nil {
		return err
	}
	if err = os.Rename(filePath+".tmp", filePath); err != nil {
		return err
	}
	return out.Close()
}

// AddFile starts file download and return name of file for fileinfo requests
func (mgr *EServerManager) AddFile(url string) (string, error) {
	log.Println("Starting download of image from ", url)
	filePath := filepath.Join(mgr.Dir, path.Base(url))
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		log.Println("file already exists ", filePath)
	} else {
		go func() {
			http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
			if err := downloadFile(filePath, url); err != nil {
				log.Fatal(err)
			}
			log.Println("Download done for ", url)
		}()
	}
	return path.Base(url), nil
}

// AddFileFromMultipart adds file from multipart.Part and returns information
func (mgr *EServerManager) AddFileFromMultipart(part *multipart.Part) *api.FileInfo {
	result := &api.FileInfo{ISReady: false}
	log.Println("Starting copy image from ", part.FileName())
	filePath := filepath.Join(mgr.Dir, part.FileName())
	if err := os.MkdirAll(filepath.Dir(filePath), os.ModeDir); err != nil {
		log.Println("cannot create dir for ", filePath)
		result.Error = err.Error()
		return result
	}
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		log.Println("file already exists ", filePath)
		// remove file if exists, we have new file in request
		if err := os.Remove(filePath); err != nil {
			result.Error = err.Error()
			return result
		}
	}
	filePathTemp := filepath.Join(mgr.Dir, fmt.Sprintf("%s.tmp", part.FileName()))
	out, err := os.Create(filePathTemp)
	if err != nil {
		result.Error = err.Error()
		return result
	}
	defer out.Close()
	hash := sha256.New()
	_, err = io.Copy(hash, io.TeeReader(part, out))
	if err != nil {
		result.Error = err.Error()
		return result
	}
	if err = os.WriteFile(fmt.Sprintf("%s.sha256", filePath), []byte(hex.EncodeToString(hash.Sum(nil))), 0666); err != nil {
		result.Error = err.Error()
		return result
	}
	if err = os.Rename(filePathTemp, filePath); err != nil {
		result.Error = err.Error()
		return result
	}
	return mgr.GetFileInfo(part.FileName())
}

// GetFileInfo checks status of file and returns information
func (mgr *EServerManager) GetFileInfo(name string) *api.FileInfo {
	result := &api.FileInfo{ISReady: false}
	filePath := filepath.Join(mgr.Dir, name)
	filePathTMP := filePath + ".tmp"
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		if _, err := os.Stat(filePathTMP); os.IsNotExist(err) {
			result.Error = err.Error()
			return result
		}
		fileSize := getFileSize(filePathTMP)
		return &api.FileInfo{
			Size:    fileSize,
			ISReady: false,
		}
	}
	fileSize := getFileSize(filePath)
	sha, err := os.ReadFile(fmt.Sprintf("%s.sha256", filePath))
	if err != nil {
		result.Error = err.Error()
		return result
	}
	return &api.FileInfo{
		Sha256:   string(sha),
		Size:     fileSize,
		FileName: path.Join("eserver", name),
		ISReady:  true,
	}
}

// GetFilePath returns path to file for serve
func (mgr *EServerManager) GetFilePath(name string) (string, error) {
	filePath := filepath.Join(mgr.Dir, name)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return "", err
	}
	return filePath, nil
}
