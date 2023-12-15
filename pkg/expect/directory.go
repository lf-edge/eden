package expect

import (
	"fmt"
	"path/filepath"

	"github.com/lf-edge/eden/pkg/utils"
	"github.com/lf-edge/eve-api/go/config"
	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
)

// createImageFile uploads image into local registry from directory
func (exp *AppExpectation) createImageDirectory(id uuid.UUID, dsID string) *config.Image {
	hash, err := utils.SHA256SUMAll(exp.appURL)
	if err != nil {
		log.Fatalf("createImageDirectory SHA256SUMAll: %v", err)
	}
	tag := fmt.Sprintf("eden/%s:%s", filepath.Base(exp.appURL), hash)
	if err := utils.CreateImage(exp.appURL, tag, exp.ctrl.GetVars().ZArch); err != nil {
		log.Fatalf("createImageDirectory CreateImage: %v", err)
	}
	if _, err := utils.LoadRegistry(tag, fmt.Sprintf("%s:%s", exp.ctrl.GetVars().RegistryIP, exp.ctrl.GetVars().RegistryPort)); err != nil {
		log.Fatalf("createImageDirectory LoadRegistry: %s", err)
	}
	return &config.Image{
		Uuidandversion: &config.UUIDandVersion{
			Uuid:    id.String(),
			Version: "1",
		},
		Name:    tag,
		Iformat: config.Format_CONTAINER,
		DsId:    dsID,
	}
}

// checkDataStoreDirectory checks if provided ds match expectation
func (exp *AppExpectation) checkDataStoreDirectory(ds *config.DatastoreConfig) bool {
	if ds.DType == config.DsType_DsContainerRegistry {
		if ds.Fqdn == fmt.Sprintf("docker://%s:%s", exp.ctrl.GetVars().RegistryIP, exp.ctrl.GetVars().RegistryPort) {
			return true
		}
	}
	return false
}

// createDataStoreDirectory creates datastore, pointed onto local registry
func (exp *AppExpectation) createDataStoreDirectory(id uuid.UUID) *config.DatastoreConfig {
	ds := &config.DatastoreConfig{
		Id:         id.String(),
		DType:      config.DsType_DsContainerRegistry,
		ApiKey:     "",
		Password:   "",
		Dpath:      "",
		Region:     "",
		CipherData: nil,
		Fqdn:       fmt.Sprintf("docker://%s:%s", exp.ctrl.GetVars().RegistryIP, exp.ctrl.GetVars().RegistryPort),
	}
	if exp.datastoreOverride != "" {
		ds.Fqdn = exp.datastoreOverride
	}
	return ds
}
