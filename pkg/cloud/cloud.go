package cloud

import (
	"github.com/lf-edge/eve/api/go/config"
	uuid "github.com/satori/go.uuid"
)

type CloudCtx struct {
	datastores       []*config.DatastoreConfig
	images           []*config.Image
	drives           map[uuid.UUID]*config.Drive
	baseOS           []*config.BaseOSConfig
	networkInstances []*config.NetworkInstanceConfig
}
