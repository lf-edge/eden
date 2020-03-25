package cloud

import (
	"errors"
	"fmt"
	"github.com/lf-edge/eve/api/go/config"
)

func (cfg *CloudCtx) AddImage(imageConfig *config.Image) error {
	for _, image := range cfg.images {
		if image.Uuidandversion.Uuid == imageConfig.Uuidandversion.GetUuid() {
			return errors.New(fmt.Sprintf("already exists with ID: %s", imageConfig.Uuidandversion.GetUuid()))
		}
	}
	_, err := cfg.GetDataStore(imageConfig.DsId)
	if err != nil {
		return err
	}
	cfg.images = append(cfg.images, imageConfig)
	return nil
}

func (cfg *CloudCtx) GetImage(Id string) (image *config.Image, err error) {
	for _, image := range cfg.images {
		if image.Uuidandversion.Uuid == Id {
			return image, nil
		}
	}
	return nil, errors.New(fmt.Sprintf("not found with ID: %s", Id))
}
