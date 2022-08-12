package utils

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/lf-edge/eden/pkg/defaults"
	log "github.com/sirupsen/logrus"
)

func CreateUsbNetConfImg(jsonConfigPath, outputImagePath string) error {
	dir, err := ioutil.TempDir("", "usb-netconf")
	if err != nil {
		err = fmt.Errorf("failed to create temporary directory: %v", err)
		return err
	}
	defer os.RemoveAll(dir)
	err = CopyFile(jsonConfigPath, filepath.Join(dir, "usb.json"))
	if err != nil {
		err = fmt.Errorf("failed to copy json config for USB image: %v", err)
		return err
	}
	// XXX Later we can also create /dump and /identity and include them in testing.
	f, err := os.Create(outputImagePath)
	if err != nil {
		err = fmt.Errorf("failed to create file %s: %v", outputImagePath, err)
		return err
	}
	if err = f.Truncate(8e6); err != nil {
		err = fmt.Errorf("failed to set size of %s to 8MB: %v", outputImagePath, err)
		return err
	}
	builder := defaults.DefaultMkimageContainerRef + ":" + defaults.DefaultMkimageTag
	u, err := RunDockerCommand(builder, "/output.img usb_conf",
		map[string]string{"/output.img": outputImagePath, "/parts": dir})
	log.Debug(u)
	if err != nil {
		err = fmt.Errorf("failed to build USB image using %s: %v",
			builder, err)
		return err
	}
	return nil
}
