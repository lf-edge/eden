package expect

import (
	"fmt"

	"github.com/lf-edge/eve/api/go/config"
	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
)

//checkImage checks if provided img match expectation
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

//createImage creates Image with provided dsID for AppExpectation
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
	default:
		actual = defaultFormat
	}
	return actual
}

//Image expects image in controller
//it gets Image with defined in AppExpectation params, or creates new one, if not exists
func (exp *AppExpectation) Image() (image *config.Image) {
	datastore := exp.DataStore()
	var err error
	for _, img := range exp.ctrl.ListImage() {
		if exp.checkImage(img, datastore.Id) {
			image = img
			break
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
