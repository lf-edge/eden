package expect

import (
	"fmt"

	"github.com/lf-edge/eve-api/go/config"
	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
)

// checkImage checks if provided img match expectation
func (exp *AppExpectation) checkImage(img *config.Image, dsID string) bool {
	if img == nil {
		return false
	}
	switch exp.appType {
	case dockerApp:
		return exp.checkImageDocker(img, dsID)
	case httpApp, httpsApp, fileApp:
		return exp.checkImageHTTP(img, dsID)
	}
	return false
}

// createImage creates Image with provided dsID for AppExpectation
func (exp *AppExpectation) createImage(dsID string) (*config.Image, error) {
	id, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}
	switch exp.appType {
	case dockerApp:
		return exp.createImageDocker(id, dsID), nil
	case httpApp, httpsApp:
		return exp.createImageHTTP(id, dsID), nil
	case fileApp:
		return exp.createImageFile(id, dsID), nil
	case directoryApp:
		return exp.createImageDirectory(id, dsID), nil
	default:
		return nil, fmt.Errorf("not supported appType")
	}
}

// imageFormatEnum return the correct enum for the image format
func (exp *AppExpectation) imageFormatEnum() config.Format {
	var defaultFormat, actual config.Format
	switch exp.appType {
	case dockerApp:
		defaultFormat = config.Format_CONTAINER
	case httpApp, httpsApp, fileApp:
		defaultFormat = config.Format_QCOW2
	default:
		defaultFormat = config.Format_QCOW2
	}
	switch exp.imageFormat {
	case "container", "oci":
		actual = config.Format_CONTAINER
	case "qcow2":
		actual = config.Format_QCOW2
	case "raw":
		actual = config.Format_RAW
	case "qcow":
		actual = config.Format_QCOW
	case "vmdk":
		actual = config.Format_VMDK
	case "vhdx":
		actual = config.Format_VHDX
	case "iso":
		actual = config.Format_ISO
	default:
		actual = defaultFormat
	}
	return actual
}

// Image expects image in controller
// it gets Image with defined in AppExpectation params, or creates new one, if not exists
func (exp *AppExpectation) Image() (image *config.Image) {
	datastore := exp.DataStore()
	var err error
	for _, appID := range exp.device.GetApplicationInstances() {
		app, err := exp.ctrl.GetApplicationInstanceConfig(appID)
		if err != nil {
			log.Fatalf("no app %s found in controller: %s", appID, err)
		}
		for _, drive := range app.Drives {
			if exp.checkImage(drive.Image, datastore.Id) {
				image = drive.Image
				break
			}
		}
	}
	for _, baseID := range exp.device.GetBaseOSConfigs() {
		base, err := exp.ctrl.GetBaseOSConfig(baseID)
		if err != nil {
			log.Fatalf("no baseOS %s found in controller: %s", baseID, err)
		}
		for _, drive := range base.Drives {
			if exp.checkImage(drive.Image, datastore.Id) {
				image = drive.Image
				break
			}
		}
	}
	if image == nil { //if image not exists, create it
		if image, err = exp.createImage(datastore.Id); err != nil {
			log.Fatalf("cannot create image: %s", err)
		}
		if err = exp.ctrl.AddImage(image); err != nil {
			log.Fatalf("AddImage: %s", err)
		}
		log.Debugf("new image created %s", image.Uuidandversion.Uuid)
	}
	return
}
