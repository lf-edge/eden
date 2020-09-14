package loaders

import (
	"github.com/lf-edge/eden/pkg/controller/cachers"
	"github.com/lf-edge/eden/pkg/controller/types"
	uuid "github.com/satori/go.uuid"
	"time"
)

//Loader interface fo controller
type Loader interface {
	SetUUID(devUUID uuid.UUID)
	ProcessStream(process ProcessFunction, typeToProcess types.LoaderObjectType, timeoutSeconds time.Duration) error
	ProcessExisting(process ProcessFunction, typeToProcess types.LoaderObjectType) error
	SetRemoteCache(cache cachers.CacheProcessor)
	Clone() Loader
}

//ProcessFunction is prototype of processing function
type ProcessFunction func(bytes []byte) (bool, error)
