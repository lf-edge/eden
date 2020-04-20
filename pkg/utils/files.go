package utils

import (
	"archive/tar"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
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

//ExtractFilesFromDocker extract all files from docker layer into directory
//if prefixDirectory is not empty, remove it from path
func ExtractFilesFromDocker(u io.ReadCloser, directory string, prefixDirectory string) error {
	pathBuilder := func(oldPath string) string {
		return path.Join(directory, strings.TrimPrefix(oldPath, prefixDirectory))
	}
	if err := os.MkdirAll(directory, 0755); err != nil {
		return fmt.Errorf("ExtractFilesFromDocker: MkdirAll() failed: %s", err.Error())
	}
	tarReader := tar.NewReader(u)
	for true {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("ExtractFilesFromDocker: Next() failed: %s", err.Error())
		}
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(pathBuilder(header.Name), 0755); err != nil {
				return fmt.Errorf("ExtractFilesFromDocker: Mkdir() failed: %s", err.Error())
			}
		case tar.TypeReg:
			if _, err := os.Lstat(pathBuilder(header.Name)); err == nil {
				err = os.Remove(pathBuilder(header.Name))
				if err != nil {
					return fmt.Errorf("ExtractFilesFromDocker: cannot remove old file: %s", err.Error())
				}
			}
			outFile, err := os.Create(pathBuilder(header.Name))
			if err != nil {
				return fmt.Errorf("ExtractFilesFromDocker: Create() failed: %s", err.Error())
			}
			if _, err := io.Copy(outFile, tarReader); err != nil {
				return fmt.Errorf("ExtractFilesFromDocker: Copy() failed: %s", err.Error())
			}
			if err := outFile.Close(); err != nil {
				return fmt.Errorf("ExtractFilesFromDocker: outFile.Close() failed: %s", err.Error())
			}
		case tar.TypeSymlink:
			if _, err := os.Lstat(pathBuilder(header.Name)); err == nil {
				err = os.Remove(pathBuilder(header.Name))
				if err != nil {
					return fmt.Errorf("ExtractFilesFromDocker: cannot remove old symlink: %s", err.Error())
				}
			}
			if err := os.Symlink(pathBuilder(header.Linkname), pathBuilder(header.Name)); err != nil {
				return fmt.Errorf("ExtractFilesFromDocker: Symlink(%s, %s) failed: %s",
					pathBuilder(header.Name), pathBuilder(header.Linkname), err.Error())
			}
		default:
			return fmt.Errorf(
				"ExtractFilesFromDocker: uknown type: '%s' in %s",
				string([]byte{header.Typeflag}),
				header.Name)
		}
	}
	return nil
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
