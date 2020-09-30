package expect

import (
	"github.com/lf-edge/eve/api/go/config"
	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
)

//imageToContentTree converts image with displayName into ContentTree representation
func (exp *appExpectation) imageToContentTree(image *config.Image, displayName string) *config.ContentTree {
	for _, el := range exp.ctrl.ListContentTree() {
		if el.URL == image.Name && el.Sha256 == image.Sha256 && el.DsId == image.DsId {
			//we already have it in controller
			return el
		}
	}
	id, err := uuid.NewV4()
	if err != nil {
		log.Fatal(err)
	}
	contentTree := &config.ContentTree{
		Uuid:            id.String(),
		DsId:            image.DsId,
		URL:             image.Name,
		Iformat:         image.Iformat,
		Sha256:          image.Sha256,
		MaxSizeBytes:    uint64(image.SizeBytes),
		DisplayName:     displayName,
		GenerationCount: 0,
	}
	_ = exp.ctrl.AddContentTree(contentTree)
	return contentTree
}
