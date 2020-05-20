package cachers

import (
	uuid "github.com/satori/go.uuid"
)

type Cacher interface {
	CheckAndSave(uuid.UUID, int, []byte) error
}

type infoOrLogs int

//LogsType for observe logs
var LogsType infoOrLogs = 1

//InfoType for observe info
var InfoType infoOrLogs = 2
