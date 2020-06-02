package expect

import (
	"fmt"
	"github.com/lf-edge/eve/api/go/config"
	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
)

func checkDataStore(ds *config.DatastoreConfig, appType appType, appUrl string) bool {
	if ds == nil {
		return false
	}
	if appType == dockerApp && ds.DType == config.DsType_DsContainerRegistry && ds.Fqdn == "docker://docker.io" {
		return true
	}
	return false
}

func createDataStore(appType appType, appUrl string) (*config.DatastoreConfig, error) {
	var ds *config.DatastoreConfig
	id, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}
	switch appType {
	case dockerApp:
		ds = &config.DatastoreConfig{
			Id:         id.String(),
			DType:      config.DsType_DsContainerRegistry,
			Fqdn:       "docker://docker.io",
			ApiKey:     "",
			Password:   "",
			Dpath:      "",
			Region:     "",
			CipherData: nil,
		}
		return ds, nil
	default:
		return nil, fmt.Errorf("not supported appType")
	}
}

//DataStore expects datastore in controller
func (exp *appExpectation) DataStore() (datastore *config.DatastoreConfig) {
	var err error
	for _, ds := range exp.ctrl.ListDataStore() {
		if checkDataStore(ds, exp.appType, exp.appUrl) {
			datastore = ds
			break
		}
	}
	if datastore == nil {
		if datastore, err = createDataStore(exp.appType, exp.appUrl); err != nil {
			log.Fatalf("cannot create datastore: %s", err)
		}
		if err = exp.ctrl.AddDataStore(datastore); err != nil {
			log.Fatalf("AddDataStore: %s", err)
		}
		log.Infof("new datastore created %s", datastore.Id)
	}
	return
}
