package controller

import (
	"github.com/lf-edge/eden/pkg/controller/einfo"
	"github.com/lf-edge/eden/pkg/controller/elog"
	"github.com/lf-edge/eden/pkg/controller/types"
	"github.com/lf-edge/eden/pkg/utils"
	uuid "github.com/satori/go.uuid"
	"time"
)

//Controller is an interface of controller
type Controller interface {
	ConfigGet(devUUID uuid.UUID) (out string, err error)
	ConfigSet(devUUID uuid.UUID, devConfig []byte) (err error)
	LogChecker(devUUID uuid.UUID, q map[string]string, handler elog.HandlerFunc, mode elog.LogCheckerMode, timeout time.Duration) (err error)
	LogLastCallback(devUUID uuid.UUID, q map[string]string, handler elog.HandlerFunc) (err error)
	InfoChecker(devUUID uuid.UUID, q map[string]string, infoType einfo.ZInfoType, handler einfo.HandlerFunc, mode einfo.InfoCheckerMode, timeout time.Duration) (err error)
	InfoLastCallback(devUUID uuid.UUID, q map[string]string, infoType einfo.ZInfoType, handler einfo.HandlerFunc) (err error)
	OnBoardList() (out []string, err error)
	DeviceList(types.DeviceStateFilter) (out []string, err error)
	DeviceGetByOnboard(eveCert string) (devUUID uuid.UUID, err error)
	DeviceGetOnboard(devUUID uuid.UUID) (onboardUUID uuid.UUID, err error)
	Register(eveCert string, eveSerial string) error
	GetDir() (dir string)
	InitWithVars(vars *utils.ConfigVars) error
}
