package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// SHA256SUM calculates sha256 of file
func SHA256SUM(filePath string) string {
	file, err := os.Open(filePath)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	hash := sha256.New()
	if _, err = io.Copy(hash, file); err != nil {
		log.Fatal(err)
	}

	return hex.EncodeToString(hash.Sum(nil))
}

// CopyFileNotExists copy file from src to dst with same permission if not exists
func CopyFileNotExists(src string, dst string) (err error) {
	if _, err = os.Lstat(dst); os.IsNotExist(err) {
		if err = CopyFile(src, dst); err != nil {
			return err
		}
	}
	return nil
}

// CopyFile copy file from src to dst with same permission
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
	if info.Mode()&os.ModeSymlink != 0 {
		//follow symlinks
		src, err = os.Readlink(src)
		if err != nil {
			return err
		}
		src = filepath.Join(filepath.Dir(src), filepath.Base(src))
	}
	in, err := os.Open(src)
	if err != nil {
		return
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return
	}
	defer func() {
		cerr := out.Close()
		if err == nil {
			err = cerr
		}
	}()
	if _, err = io.Copy(out, in); err != nil {
		return
	}
	err = out.Sync()
	return
}

// TouchFile create empty file
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

// FileNameWithoutExtension trim file extension and path
func FileNameWithoutExtension(fileName string) string {
	return filepath.Base(strings.TrimSuffix(fileName, filepath.Ext(fileName)))
}

// ResolveAbsPath use eden.root parameter to resolve path
func ResolveAbsPath(curPath string) string {
	return ResolveAbsPathWithRoot(viper.GetString("eden.root"), curPath)
}

// ResolveAbsPathWithRoot use rootPath parameter to resolve path
func ResolveAbsPathWithRoot(rootPath, curPath string) string {
	if strings.TrimSpace(curPath) == "" {
		return ""
	}
	if !filepath.IsAbs(curPath) {
		return filepath.Join(rootPath, strings.TrimSpace(curPath))
	}
	return curPath
}

// GetFileFollowLinks resolve file by walking through symlinks
func GetFileFollowLinks(filePath string) (string, error) {
	log.Debugf("GetFileFollowLinks %s", filePath)
	filePath = ResolveHomeDir(filePath)
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

// GetFileSize returns file size
func GetFileSize(filePath string) int64 {
	fi, err := os.Stat(filePath)
	if err != nil {
		log.Fatal(err)
	}
	return fi.Size()
}

// ResolveHomeDir resolve ~ in path
func ResolveHomeDir(filePath string) string {
	usr, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}
	dir := usr.HomeDir
	if filePath == "~" {
		filePath = dir
	} else if strings.HasPrefix(filePath, "~/") {
		filePath = filepath.Join(dir, filePath[2:])
	}
	return filePath
}

// CopyFolder from source to destination
func CopyFolder(source, destination string) error {
	var err = filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		var relPath = strings.Replace(path, source, "", 1)
		if relPath == "" {
			return nil
		}
		if info.IsDir() {
			return os.Mkdir(filepath.Join(destination, relPath), info.Mode())
		}
		return CopyFile(filepath.Join(source, relPath), filepath.Join(destination, relPath))
	})
	return err
}

// IsInputFromPipe returns true if the command is running from pipe
func IsInputFromPipe() bool {
	fileInfo, _ := os.Stdin.Stat()
	return fileInfo.Mode()&os.ModeCharDevice == 0
}

// SHA256SUMAll calculates sha256 of directory
func SHA256SUMAll(dir string) (string, error) {
	hash := sha256.New()
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		r, err := os.Open(path)
		if err != nil {
			return err
		}
		if _, err := io.Copy(hash, r); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

// CreateDisk creates empty disk with defined format on diskFile with size bytes capacity
func CreateDisk(diskFile, format string, size uint64) error {
	if err := os.MkdirAll(filepath.Dir(diskFile), 0755); err != nil {
		return err
	}
	return RunCommandForeground("qemu-img", "create", "-f", format, diskFile, fmt.Sprintf("%d", size))
}
