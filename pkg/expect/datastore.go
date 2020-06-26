package expect

import (
	"fmt"
	"github.com/lf-edge/eve/api/go/config"
	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
)

func (exp *appExpectation) checkDataStore(ds *config.DatastoreConfig) bool {
	if ds == nil {
		return false
	}
	switch exp.appType {
	case dockerApp:
		return exp.checkDataStoreDocker(ds)
	case httpApp, httpsApp:
		return exp.checkDataStoreHttp(ds)
	}
	return false
}

func (exp *appExpectation) createDataStore() (*config.DatastoreConfig, error) {
	id, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}
	switch exp.appType {
	case dockerApp:
		return exp.createDataStoreDocker(id), nil
	case httpApp, httpsApp:
		return exp.createDataStoreHttp(id), nil
	default:
		return nil, fmt.Errorf("not supported appType")
	}
}

//DataStore expects datastore in controller
func (exp *appExpectation) DataStore() (datastore *config.DatastoreConfig) {
	var err error
	for _, ds := range exp.ctrl.ListDataStore() {
		if exp.checkDataStore(ds) {
			datastore = ds
			break
		}
	}
	if datastore == nil {
		if datastore, err = exp.createDataStore(); err != nil {
			log.Fatalf("cannot create datastore: %s", err)
		}
		if err = exp.ctrl.AddDataStore(datastore); err != nil {
			log.Fatalf("AddDataStore: %s", err)
		}
		log.Infof("new datastore created %s", datastore.Id)
	}
	return
}
