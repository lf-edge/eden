package utils

import (
	"bytes"
	"fmt"
	"github.com/lf-edge/eden/pkg/defaults"
	uuid "github.com/satori/go.uuid"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"text/template"
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

// Image extracts image tag from EVEDescription
func (desc EVEDescription) Image() (string, error) {
	if desc.Registry == "" {
		desc.Registry = defaults.DefaultEveRegistry
	}
	version, err := desc.Version()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s:%s", desc.Registry, version), nil
}

// Version extracts version from EVEDescription
func (desc EVEDescription) Version() (string, error) {
	if desc.Tag == "" {
		return "", fmt.Errorf("tag not present")
	}
	if desc.Arch == "" {
		return "", fmt.Errorf("arch not present")
	}
	if desc.HV == "" {
		return "", fmt.Errorf("hv not present")
	}
	return fmt.Sprintf("%s-%s-%s", desc.Tag, desc.HV, desc.Arch), nil
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
		return fmt.Sprintf("%s-uefi:latest-%s", defaults.DefaultEveRegistry, desc.Arch), nil
	}
	if desc.Tag == "" {
		return "", fmt.Errorf("tag not present")
	}
	if desc.Arch == "" {
		return "", fmt.Errorf("arch not present")
	}
	return fmt.Sprintf("%s-uefi:%s-%s", desc.Registry, desc.Tag, desc.Arch), nil
}

//DownloadEveInstaller pulls EVE installer image from docker
func DownloadEveInstaller(eve EVEDescription, outputFile string) (err error) {
	image, err := eve.Image()
	if err != nil {
		return err
	}
	fileName, err := genEVEInstallerImage(image, filepath.Dir(outputFile), eve.ConfigPath)
	if err != nil {
		return fmt.Errorf("genEVEImage: %s", err)
	}
	if err = CopyFile(fileName, outputFile); err != nil {
		return fmt.Errorf("cannot copy image %s", err)
	}
	return nil
}

func DownloadUEFI(uefi UEFIDescription, outputDir string) (err error) {
	efiImage, err := uefi.image(false) //download OVMF
	if err != nil {
		return err
	}
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
	if err := SaveImageAndExtract(efiImage, outputDir, ""); err != nil {
		return fmt.Errorf("SaveImage: %s", err)
	}
	return nil
}

//DownloadEveLive pulls EVE live image from docker
func DownloadEveLive(eve EVEDescription, outputFile string) (err error) {
	image, err := eve.Image()
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
	}
	if eve.Format == "gcp" || eve.Format == "vdi" || eve.Format == "parallels" {
		size = eve.ImageSizeMB
	}
	fileName, err := genEVELiveImage(image, filepath.Dir(outputFile), eve.Format, eve.ConfigPath, size)
	if err != nil {
		return fmt.Errorf("genEVEImage: %s", err)
	}
	if eve.Format == "parallels" {
		dirForParallels := strings.TrimRight(outputFile, filepath.Ext(outputFile))
		_ = os.Mkdir(dirForParallels, 0777)
		if err = CopyFile(fileName, filepath.Join(dirForParallels, fmt.Sprintf("live.0.%s.hds", defaults.DefaultParallelsUUID))); err != nil {
			return fmt.Errorf("cannot copy image %s", err)
		}

		t := template.New("t")
		t, err := t.Parse(defaults.ParallelsDiskTemplate)
		if err != nil {
			log.Fatal(err)
		}
		buf := new(bytes.Buffer)
		uid, _ := uuid.NewV4()
		err = t.Execute(buf, struct {
			DiskSize    int
			Cylinders   int
			UID         string
			SnapshotUID string
		}{
			DiskSize:    eve.ImageSizeMB * 1024 * 1024 / 16 / 32,
			Cylinders:   eve.ImageSizeMB * 1024 * 1024 / 16 / 32 / 512,
			UID:         uid.String(),
			SnapshotUID: defaults.DefaultParallelsUUID,
		})
		if err != nil {
			log.Fatal(err)
		}
		if err = ioutil.WriteFile(filepath.Join(dirForParallels, "DiskDescriptor.xml"), buf.Bytes(), 0777); err != nil {
			return fmt.Errorf("cannot write description %s", err)
		}
	}
	if err = CopyFile(fileName, outputFile); err != nil {
		return fmt.Errorf("cannot copy image %s", err)
	}
	return nil
}

//genEVEInstallerImage downloads EVE installer image from docker to outputDir with configDir (if defined)
func genEVEInstallerImage(image, outputDir string, configDir string) (fileName string, err error) {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", err
	}
	volumeMap := map[string]string{"/out": outputDir}
	if configDir != "" {
		volumeMap = map[string]string{"/in": configDir, "/out": outputDir}
	}
	fileName = filepath.Join(outputDir, "installer.raw")
	dockerCommand := "-f raw installer_raw"
	u, err := RunDockerCommand(image, dockerCommand, volumeMap)
	if err != nil {
		return "", err
	}
	log.Debug(u)
	return fileName, nil
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
	if format == "vdi" {
		fileName = fileName + "." + format
	}
	if format == "parallels" {
		fileName = fileName + ".parallels"
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
	image, err := eve.Image()
	if err != nil {
		return "", err
	}
	log.Debugf("Try ImagePull with (%s)", image)
	if err := PullImage(image); err != nil {
		return "", fmt.Errorf("ImagePull (%s): %s", image, err)
	}
	var size int
	fileName, err := genEVERootFSImage(eve, outputDir, size)
	if err != nil {
		return "", fmt.Errorf("genEVEImage: %s", err)
	}
	return filepath.Join(outputDir, filepath.Base(fileName)), nil
}

//genEVERootFSImage downloads EVE rootfs image from docker to outputDir
func genEVERootFSImage(eve EVEDescription, outputDir string, size int) (fileName string, err error) {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", err
	}
	image, err := eve.Image()
	if err != nil {
		return "", err
	}
	version, err := eve.Version()
	if err != nil {
		return "", err
	}
	log.Debug("Try to get version of rootfs")
	dockerCommand := "-f raw version"
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
	log.Debugf("image version: %s", u)
	log.Debugf("provided version: %s", version)
	fileName = filepath.Join(outputDir, fmt.Sprintf("rootfs-%s.squashfs", version))
	correctionFileName := filepath.Join(outputDir, fmt.Sprintf("rootfs-%s.squashfs.ver", version))
	_ = os.Remove(correctionFileName)
	volumeMap := map[string]string{"/out": outputDir}
	dockerCommand = fmt.Sprintf("-f raw rootfs %d", size)
	if size == 0 {
		dockerCommand = "-f raw rootfs"
	}
	cmdOut, err := RunDockerCommand(image, dockerCommand, volumeMap)
	if err != nil {
		return "", err
	}
	log.Debug(cmdOut)
	if err = CopyFile(filepath.Join(outputDir, "rootfs.img"), fileName); err != nil {
		return "", err
	}
	if version != u {
		log.Warningf("Versions mismatch (loaded %s vs provided %s): write correction file %s",
			u, version, correctionFileName)
		if err := ioutil.WriteFile(correctionFileName, []byte(u), 0755); err != nil {
			return "", err
		}
	}
	return fileName, nil
}

//DownloadEveNetBoot pulls EVE image from docker and prepares files for net boot
func DownloadEveNetBoot(eve EVEDescription, outputDir string) (err error) {
	image, err := eve.Image()
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

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return err
	}
	volumeMap := map[string]string{"/out": outputDir}
	if eve.ConfigPath != "" {
		volumeMap = map[string]string{"/in": eve.ConfigPath, "/out": outputDir}
	}
	dockerCommand := "installer_net"
	u, err := RunDockerCommand(image, dockerCommand, volumeMap)
	if err != nil {
		return err
	}
	log.Debug(u)
	tempFile := filepath.Join(outputDir, "installer.net")
	if err := Untar(tempFile, outputDir); err != nil {
		log.Fatalf("Untar: %s", err)
	}
	return os.Remove(tempFile)
}
