package cachers

import (
	"github.com/lf-edge/eden/pkg/controller/types"
	uuid "github.com/satori/go.uuid"
)

//CacheProcessor for processing objects and save into cache
type CacheProcessor interface {
	CheckAndSave(uuid.UUID, types.LoaderObjectType, []byte) error
}
