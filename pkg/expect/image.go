package expect

import (
	"fmt"
	"github.com/lf-edge/eve/api/go/config"
	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
)

func checkImage(img *config.Image, dsId string, appType appType, appUrl string, appVersion string) bool {
	if img == nil {
		return false
	}
	if appType == dockerApp {
		if img.DsId == dsId && img.Name == fmt.Sprintf("%s:%s", appUrl, appVersion) && img.Iformat == config.Format_CONTAINER {
			return true
		}
	}
	return false
}

func createImage(dsId string, appType appType, appUrl string, appVersion string) (*config.Image, error) {
	var img *config.Image
	id, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}
	switch appType {
	case dockerApp:
		img = &config.Image{
			Uuidandversion: &config.UUIDandVersion{
				Uuid:    id.String(),
				Version: "1",
			},
			Name:    fmt.Sprintf("%s:%s", appUrl, appVersion),
			Iformat: config.Format_CONTAINER,
			DsId:    dsId,
			Siginfo: &config.SignatureInfo{},
		}
		return img, nil
	default:
		return nil, fmt.Errorf("not supported appType")
	}
}

//Image expects image in controller
func (exp *appExpectation) Image() (image *config.Image) {
	datastore := exp.DataStore()
	var err error
	for _, img := range exp.ctrl.ListImage() {
		if checkImage(img, datastore.Id, exp.appType, exp.appUrl, exp.appVersion) {
			image = img
			break
		}
	}
	if image == nil {
		if image, err = createImage(datastore.Id, exp.appType, exp.appUrl, exp.appVersion); err != nil {
			log.Fatalf("cannot create image: %s", err)
		}
		if err = exp.ctrl.AddImage(image); err != nil {
			log.Fatalf("AddImage: %s", err)
		}
		log.Infof("new image created %s", image.Uuidandversion.Uuid)
	}
	return
}
