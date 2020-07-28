package utils

import (
	"fmt"
	"github.com/lf-edge/eden/pkg/defaults"
	log "github.com/sirupsen/logrus"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

//DownloadEveLive pulls EVE live image from docker
func DownloadEveLive(configPath string, outputFile string, eveArch string, eveHV string, eveTag string, format string) (err error) {
	efiImage := fmt.Sprintf("lfedge/eve-uefi:%s-%s", eveTag, eveArch) //download OVMF
	image := fmt.Sprintf("lfedge/eve:%s-%s-%s", eveTag, eveHV, eveArch)
	log.Debugf("Try ImagePull with (%s)", image)
	if err := PullImage(image); err != nil {
		return fmt.Errorf("ImagePull (%s): %s", image, err)
	}
	if configPath != "" {
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			return fmt.Errorf("directory not exists: %s", configPath)
		}
	}
	size := 0
	if format == "qcow2" {
		format = "qcow2"
		size = defaults.DefaultEVEImageSize
		if err := PullImage(efiImage); err != nil {
			log.Infof("cannot pull %s", efiImage)
			efiImage = fmt.Sprintf("lfedge/eve-uefi") //try with latest version of OVMF
			log.Infof("will retry with %s", efiImage)
			if err := PullImage(efiImage); err != nil {
				return fmt.Errorf("ImagePull (%s): %s", efiImage, err)
			}
		}
		if err := SaveImage(efiImage, filepath.Dir(outputFile), ""); err != nil {
			return fmt.Errorf("SaveImage: %s", err)
		}
	}
	if fileName, err := genEVELiveImage(image, filepath.Dir(outputFile), format, configPath, size); err != nil {
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
func DownloadEveRootFS(outputDir string, eveArch string, eveHV string, eveTag string) (filePath string, err error) {
	image := fmt.Sprintf("lfedge/eve:%s-%s-%s", eveTag, eveHV, eveArch)
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
