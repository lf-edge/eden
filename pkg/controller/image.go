package controller

import (
	"fmt"
	"github.com/lf-edge/eve/api/go/config"
)

//GetImage return Image config from cloud by ID
func (cloud *CloudCtx) GetImage(id string) (image *config.Image, err error) {
	for _, image := range cloud.images {
		if image.Uuidandversion.Uuid == id {
			return image, nil
		}
	}
	return nil, fmt.Errorf("not found Image with ID: %s", id)
}

//AddImage add Image config to cloud
func (cloud *CloudCtx) AddImage(imageConfig *config.Image) error {
	for _, image := range cloud.images {
		if image.Uuidandversion.Uuid == imageConfig.Uuidandversion.GetUuid() {
			return fmt.Errorf("image already exists with ID: %s", imageConfig.Uuidandversion.GetUuid())
		}
	}
	_, err := cloud.GetDataStore(imageConfig.DsId)
	if err != nil {
		return err
	}
	cloud.images = append(cloud.images, imageConfig)
	return nil
}
