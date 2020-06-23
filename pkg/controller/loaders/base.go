package loaders

import (
	"github.com/lf-edge/eden/pkg/controller/cachers"
	uuid "github.com/satori/go.uuid"
	"time"
)

//Loader interface fo controller
type Loader interface {
	SetUUID(devUUID uuid.UUID)
	ProcessStream(process ProcessFunction, typeToProcess infoOrLogs, timeoutSeconds time.Duration) error
	ProcessExisting(process ProcessFunction, typeToProcess infoOrLogs) error
	SetRemoteCache(cache cachers.Cacher)
	Clone() Loader
}

type infoOrLogs int

//LogsType for observe logs
var LogsType infoOrLogs = 1

//InfoType for observe info
var InfoType infoOrLogs = 2

//MetricsType for observe metrics
var MetricsType infoOrLogs = 3

//ProcessFunction is prototype of processing function
type ProcessFunction func(bytes []byte) (bool, error)
