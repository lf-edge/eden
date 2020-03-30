package controller

import (
	"github.com/lf-edge/eden/pkg/controller/einfo"
	uuid "github.com/satori/go.uuid"
	"time"
)

//Controller is an interface of controller
type Controller interface {
	ConfigGet(devUUID *uuid.UUID) (out string, err error)
	ConfigSet(devUUID *uuid.UUID, devConfig []byte) (err error)
	LogChecker(devUUID *uuid.UUID, q map[string]string, timeout time.Duration) (err error)
	InfoChecker(devUUID *uuid.UUID, q map[string]string, infoType einfo.ZInfoType, timeout time.Duration) (err error)
	OnBoardList() (out []string, err error)
	DeviceList() (out []string, err error)
	Register(eveCert string, eveSerial string) error
	GetDir() (dir string)
}
