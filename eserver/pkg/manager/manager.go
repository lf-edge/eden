package manager

import (
	"crypto/tls"
	"fmt"
	"github.com/lf-edge/eden/eserver/api"
	"github.com/lf-edge/eden/pkg/utils"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"path/filepath"
)

//EServerManager for process files
type EServerManager struct {
	Dir string
}

//Init directories for EServerManager
func (mgr *EServerManager) Init() {
	if _, err := os.Stat(mgr.Dir); err != nil {
		if err = os.MkdirAll(mgr.Dir, 0755); err != nil {
			log.Fatal(err)
		}
	}
}

//ListFileNames list downloaded files
func (mgr *EServerManager) ListFileNames() (result []string) {
	files, err := ioutil.ReadDir(mgr.Dir)
	if err != nil {
		log.Fatal(err)
	}
	for _, f := range files {
		result = append(result, f.Name())
	}
	return
}

//AddFile starts file download and return name of file for fileinfo requests
func (mgr *EServerManager) AddFile(url string) (string, error) {
	log.Println("Starting download of image from ", url)
	filePath := filepath.Join(mgr.Dir, path.Base(url))
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		log.Println("file already exists ", filePath)
	} else {
		go func() {
			http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
			if err := utils.DownloadFile(filePath, url); err != nil {
				log.Fatal(err)
			}
			log.Println("Download done for ", url)
		}()
	}
	return path.Base(url), nil
}

//AddFileFromMultipart adds file from multipart.File and returns information
func (mgr *EServerManager) AddFileFromMultipart(file multipart.File, fileHeader *multipart.FileHeader) *api.FileInfo {
	result := &api.FileInfo{ISReady: false}
	log.Println("Starting copy image from ", fileHeader.Filename)
	filePath := filepath.Join(mgr.Dir, fileHeader.Filename)
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		log.Println("file already exists ", filePath)
	} else {
		filePathTemp := filepath.Join(mgr.Dir, fmt.Sprintf("%s.tmp", fileHeader.Filename))
		out, err := os.Create(filePathTemp)
		if err != nil {
			result.Error = err.Error()
			return result
		}
		defer out.Close()
		_, err = io.Copy(out, file)
		if err != nil {
			result.Error = err.Error()
			return result
		}
		if utils.GetFileSize(filePathTemp) != fileHeader.Size {
			result.Error = "file sizes doesn't match"
			return result
		}
		if err = os.Rename(filePathTemp, filePath); err != nil {
			result.Error = err.Error()
			return result
		}
	}
	return mgr.GetFileInfo(fileHeader.Filename)
}

//GetFileInfo checks status of file and returns information
func (mgr *EServerManager) GetFileInfo(name string) *api.FileInfo {
	result := &api.FileInfo{ISReady: false}
	filePath := filepath.Join(mgr.Dir, name)
	filePathTMP := filePath + ".tmp"
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		if _, err := os.Stat(filePathTMP); os.IsNotExist(err) {
			result.Error = err.Error()
			return result
		} else {
			fileSize := utils.GetFileSize(filePathTMP)
			return &api.FileInfo{
				Size:    fileSize,
				ISReady: false,
			}
		}
	} else {
		fileSize := utils.GetFileSize(filePath)
		sha256, err := utils.SHA256SUM(filePath)
		if err != nil {
			result.Error = err.Error()
			return result
		}
		return &api.FileInfo{
			Sha256:   sha256,
			Size:     fileSize,
			FileName: path.Join("eserver", name),
			ISReady:  true,
		}
	}
}

//GetFilePath returns path to file for serve
func (mgr *EServerManager) GetFilePath(name string) (string, error) {
	filePath := filepath.Join(mgr.Dir, name)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return "", err
	} else {
		return filePath, nil
	}
}
