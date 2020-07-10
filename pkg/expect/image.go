package expect

import (
	"fmt"
	"github.com/lf-edge/eve/api/go/config"
	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
)

//checkImage checks if provided img match expectation
func (exp *appExpectation) checkImage(img *config.Image, dsId string) bool {
	if img == nil {
		return false
	}
	switch exp.appType {
	case dockerApp:
		return exp.checkImageDocker(img, dsId)
	case httpApp, httpsApp, fileApp:
		return exp.checkImageHttp(img, dsId)
	}
	return false
}

//createImage creates Image with provided dsId for appExpectation
func (exp *appExpectation) createImage(dsId string) (*config.Image, error) {
	id, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}
	switch exp.appType {
	case dockerApp:
		return exp.createImageDocker(id, dsId), nil
	case httpApp, httpsApp:
		return exp.createImageHttp(id, dsId), nil
	case fileApp:
		return exp.createImageFile(id, dsId), nil
	default:
		return nil, fmt.Errorf("not supported appType")
	}
}

//Image expects image in controller
//it gets Image with defined in appExpectation params, or creates new one, if not exists
func (exp *appExpectation) Image() (image *config.Image) {
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
		log.Infof("new image created %s", image.Uuidandversion.Uuid)
	}
	return
}
