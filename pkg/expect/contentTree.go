package expect

import (
	"github.com/lf-edge/eve-api/go/config"
	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
)

// imageToContentTree converts image with displayName into ContentTree representation
func (exp *AppExpectation) imageToContentTree(image *config.Image, displayName string) *config.ContentTree {
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

// ContentTree creates a standalone ContentTree from the AppExpectation's
// resolved image, registers it with the controller, and adds its UUID to the
// device's ContentTree list. No Volume is created. Pillar's volumemgr
// downloads ContentTrees eagerly; blob lookup is by SHA256, so a later
// AppInstance referencing the same image reuses the blobs without
// re-downloading. Use this to pre-stage content trees for tests that
// exercise upgrade/migration scenarios without involving the PVC machinery.
func (exp *AppExpectation) ContentTree(displayName string) *config.ContentTree {
	img := exp.Image()
	if displayName == "" {
		displayName = img.Name
	}
	contentTree := exp.imageToContentTree(img, displayName)
	exp.device.SetContentTreeConfig(append(exp.device.GetContentTrees(), contentTree.Uuid))
	return contentTree
}
