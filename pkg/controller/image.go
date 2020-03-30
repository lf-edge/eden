package controller

import (
	"fmt"
	"github.com/lf-edge/eve/api/go/config"
)

//GetImage return Image config from cloud by ID
func (cfg *Ctx) GetImage(ID string) (image *config.Image, err error) {
	for _, image := range cfg.images {
		if image.Uuidandversion.Uuid == ID {
			return image, nil
		}
	}
	return nil, fmt.Errorf("not found with ID: %s", ID)
}

//AddImage add Image config to cloud
func (cfg *Ctx) AddImage(imageConfig *config.Image) error {
	for _, image := range cfg.images {
		if image.Uuidandversion.Uuid == imageConfig.Uuidandversion.GetUuid() {
			return fmt.Errorf("already exists with ID: %s", imageConfig.Uuidandversion.GetUuid())
		}
	}
	_, err := cfg.GetDataStore(imageConfig.DsId)
	if err != nil {
		return err
	}
	cfg.images = append(cfg.images, imageConfig)
	return nil
}
