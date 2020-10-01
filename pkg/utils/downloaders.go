package utils

import (
	"fmt"
	"github.com/lf-edge/eden/pkg/defaults"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	log "github.com/sirupsen/logrus"
)

// EVEDescription provides information about EVE to download
type EVEDescription struct {
	ConfigPath  string
	Arch        string
	HV          string
	Registry    string
	Tag         string
	Format      string
	ImageSizeMB int
}

// image extracts image tag from EVEDescription
func (desc EVEDescription) image() (string, error) {
	if desc.Registry == "" {
		desc.Registry = defaults.DefaultEveRegistry
	}
	if desc.Tag == "" {
		return "", fmt.Errorf("tag not present")
	}
	if desc.Arch == "" {
		return "", fmt.Errorf("arch not present")
	}
	if desc.HV == "" {
		return "", fmt.Errorf("hv not present")
	}
	return fmt.Sprintf("%s/eve:%s-%s-%s", desc.Registry, desc.Tag, desc.HV, desc.Arch), nil
}

// UEFIDescription provides information about UEFI to download
type UEFIDescription struct {
	Registry string
	Tag      string
	Arch     string
}

// image extracts image tag from UEFIDescription
func (desc UEFIDescription) image(latest bool) (string, error) {
	if desc.Registry == "" {
		desc.Registry = defaults.DefaultEveRegistry
	}
	if latest {
		return fmt.Sprintf("%s/eve-uefi", desc.Registry), nil
	}
	if desc.Tag == "" {
		return "", fmt.Errorf("tag not present")
	}
	if desc.Arch == "" {
		return "", fmt.Errorf("arch not present")
	}
	return fmt.Sprintf("%s/eve-uefi:%s-%s", desc.Registry, desc.Tag, desc.Arch), nil
}

//DownloadEveLive pulls EVE live image from docker
func DownloadEveLive(eve EVEDescription, uefi UEFIDescription, outputFile string) (err error) {
	efiImage, err := uefi.image(false) //download OVMF
	if err != nil {
		return err
	}
	image, err := eve.image()
	if err != nil {
		return err
	}
	log.Debugf("Try ImagePull with (%s)", image)
	if err := PullImage(image); err != nil {
		return fmt.Errorf("ImagePull (%s): %s", image, err)
	}
	if eve.ConfigPath != "" {
		if _, err := os.Stat(eve.ConfigPath); os.IsNotExist(err) {
			return fmt.Errorf("directory not exists: %s", eve.ConfigPath)
		}
	}
	size := 0
	if eve.Format == "qcow2" {
		size = eve.ImageSizeMB
		if err := PullImage(efiImage); err != nil {
			log.Infof("cannot pull %s", efiImage)
			efiImage, err = uefi.image(true) //try with latest version of OVMF
			if err != nil {
				return err
			}
			log.Infof("will retry with %s", efiImage)
			if err := PullImage(efiImage); err != nil {
				return fmt.Errorf("ImagePull (%s): %s", efiImage, err)
			}
		}
		if err := SaveImageAndExtract(efiImage, filepath.Dir(outputFile), ""); err != nil {
			return fmt.Errorf("SaveImage: %s", err)
		}
	}
	if eve.Format == "gcp" {
		size = eve.ImageSizeMB
	}
	if fileName, err := genEVELiveImage(image, filepath.Dir(outputFile), eve.Format, eve.ConfigPath, size); err != nil {
		return fmt.Errorf("genEVEImage: %s", err)
	} else {
		if err = CopyFile(fileName, outputFile); err != nil {
			return fmt.Errorf("cannot copy image %s", err)
		}
		return nil
	}
}

//genEVELiveImage downloads EVE live image from docker to outputDir with configDir (if defined)
func genEVELiveImage(image, outputDir string, format string, configDir string, size int) (fileName string, err error) {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", err
	}
	volumeMap := map[string]string{"/out": outputDir}
	if configDir != "" {
		volumeMap = map[string]string{"/in": configDir, "/out": outputDir}
	}
	fileName = filepath.Join(outputDir, "live.raw")
	if format == "qcow2" {
		fileName = fileName + "." + format
	}
	if format == "gcp" {
		fileName = fileName + ".img.tar.gz"
	}
	dockerCommand := fmt.Sprintf("-f %s live %d", format, size)
	if size == 0 {
		dockerCommand = fmt.Sprintf("-f %s live", format)
	}
	u, err := RunDockerCommand(image, dockerCommand, volumeMap)
	if err != nil {
		return "", err
	}
	log.Debug(u)
	return fileName, nil
}

//DownloadEveRootFS pulls EVE rootfs image from docker
func DownloadEveRootFS(eve EVEDescription, outputDir string) (filePath string, err error) {
	image, err := eve.image()
	log.Debugf("Try ImagePull with (%s)", image)
	if err := PullImage(image); err != nil {
		return "", fmt.Errorf("ImagePull (%s): %s", image, err)
	}
	var size int
	if fileName, err := genEVERootFSImage(image, outputDir, size); err != nil {
		return "", fmt.Errorf("genEVEImage: %s", err)
	} else {
		return filepath.Join(outputDir, filepath.Base(fileName)), nil
	}
}

//genEVERootFSImage downloads EVE rootfs image from docker to outputDir
func genEVERootFSImage(image, outputDir string, size int) (fileName string, err error) {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", err
	}
	log.Debug("Try to get version of rootfs")
	dockerCommand := fmt.Sprintf("-f raw version")
	u, err := RunDockerCommand(image, dockerCommand, nil)
	if err != nil {
		return "", err
	}
	u = strings.TrimFunc(u, func(r rune) bool {
		if unicode.IsDigit(r) || unicode.IsLetter(r) || r == '.' || r == '-' {
			return false
		}
		return true
	})
	log.Debugf("rootfs version: %s", u)
	fileName = filepath.Join(outputDir, fmt.Sprintf("rootfs-%s.squashfs", u))
	volumeMap := map[string]string{"/out": outputDir}
	dockerCommand = fmt.Sprintf("-f raw rootfs %d", size)
	if size == 0 {
		dockerCommand = "-f raw rootfs"
	}
	u, err = RunDockerCommand(image, dockerCommand, volumeMap)
	if err != nil {
		return "", err
	}
	log.Debug(u)
	if err = CopyFile(filepath.Join(outputDir, "rootfs.img"), fileName); err != nil {
		return "", err
	}
	return fileName, nil
}
