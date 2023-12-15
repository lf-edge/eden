package expect

import (
	"fmt"

	"github.com/lf-edge/eve-api/go/config"
	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
)

// checkDataStore checks if provided ds match expectation
func (exp *AppExpectation) checkDataStore(ds *config.DatastoreConfig) bool {
	if ds == nil {
		return false
	}
	switch exp.appType {
	case dockerApp:
		return exp.checkDataStoreDocker(ds)
	case httpApp, httpsApp, fileApp:
		return exp.checkDataStoreHTTP(ds)
	case directoryApp:
		return exp.checkDataStoreDirectory(ds)
	}
	return false
}

// createDataStore creates DatastoreConfig for AppExpectation
func (exp *AppExpectation) createDataStore() (*config.DatastoreConfig, error) {
	id, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}
	switch exp.appType {
	case dockerApp:
		return exp.createDataStoreDocker(id), nil
	case httpApp, httpsApp, fileApp:
		if exp.sftpLoad {
			return exp.createDataStoreSFTP(id), nil
		}
		return exp.createDataStoreHTTP(id), nil
	case directoryApp:
		return exp.createDataStoreDirectory(id), nil
	default:
		return nil, fmt.Errorf("not supported appType")
	}
}

// DataStore expects datastore in controller
// it gets DatastoreConfig with defined in AppExpectation params, or creates new one, if not exists
func (exp *AppExpectation) DataStore() (datastore *config.DatastoreConfig) {
	var err error
	for _, ds := range exp.ctrl.ListDataStore() {
		if exp.checkDataStore(ds) {
			datastore = ds
			break
		}
	}
	if datastore == nil { //if datastore not exists, create it
		if datastore, err = exp.createDataStore(); err != nil {
			log.Fatalf("cannot create datastore: %s", err)
		}
		exp.applyDatastoreCipher(datastore)
		if err = exp.ctrl.AddDataStore(datastore); err != nil {
			log.Fatalf("AddDataStore: %s", err)
		}
		log.Debugf("new datastore created %s", datastore.Id)
	}
	return
}
