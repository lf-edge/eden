//Package einfo provides primitives for searching and processing data
//in Info files.
package einfo

import (
	"fmt"
	"github.com/golang/protobuf/jsonpb"
	"github.com/lf-edge/eden/pkg/controller/loaders"
	"github.com/lf-edge/eden/pkg/utils"
	"github.com/lf-edge/eve/api/go/info"
	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
	"reflect"
	"regexp"
	"strings"
	"time"
)

//HandlerFunc must process info.ZInfoMsg and return true to exit
//or false to continue
type HandlerFunc func(im *info.ZInfoMsg, ds []*ZInfoMsgInterface, infoType ZInfoType) bool

//QHandlerFunc must process info.ZInfoMsg with query parameters
//and return true to exit or false to continue
type QHandlerFunc func(im *info.ZInfoMsg, query map[string]string, infoType ZInfoType) []*ZInfoMsgInterface

//ZInfoMsgInterface is an interface to pass between handlers
type ZInfoMsgInterface interface{}

type zInfoPacket struct {
	upperType string
	lowerType string
}

//ZInfoType is an parameter for obtain particular info from files
type ZInfoType *zInfoPacket

var (
	//ZInfoDinfo can be used for filter GetNiinfo
	ZInfoDinfo ZInfoType = &zInfoPacket{upperType: "GetDinfo"}
	//ZInfoDevSW can be used for filter GetDinfo SwList
	ZInfoDevSW ZInfoType = &zInfoPacket{upperType: "GetDinfo", lowerType: "SwList"}
	//ZInfoNetwork can be used for filter GetDinfo Network
	ZInfoNetwork ZInfoType = &zInfoPacket{upperType: "GetDinfo", lowerType: "Network"}
	//ZInfoNetworkInstance can be used for filter GetNiinfo
	ZInfoNetworkInstance ZInfoType = &zInfoPacket{upperType: "GetNiinfo"}
	//ZInfoAppInstance can be used for filter GetAinfo
	ZInfoAppInstance ZInfoType = &zInfoPacket{upperType: "GetAinfo"}
	//ZAll can be used for display all info items
	ZAll ZInfoType = &zInfoPacket{}
)

//GetZInfoType return ZInfoType by name
func GetZInfoType(name string) (ZInfoType, error) {
	var zInfoType ZInfoType
	switch name {
	case "all":
		zInfoType = ZAll
	case "dinfo-network":
		zInfoType = ZInfoNetwork
	case "dinfo-swlist":
		zInfoType = ZInfoDevSW
	case "ainfo":
		zInfoType = ZInfoAppInstance
	case "niinfo":
		zInfoType = ZInfoNetworkInstance
	default:
		return nil, fmt.Errorf("not implemented: %s", name)
	}
	return zInfoType, nil
}

//ListZInfoType return all implemented
func ListZInfoType() []string {
	return []string{"all", "dinfo-network", "dinfo-swlist", "ainfo", "niinfo"}
}

//ParseZInfoMsg unmarshal ZInfoMsg
func ParseZInfoMsg(data []byte) (ZInfoMsg info.ZInfoMsg, err error) {
	var zi info.ZInfoMsg
	err = jsonpb.UnmarshalString(string(data), &zi)
	return zi, err
}

//InfoPrn print data from ZInfoMsg structure
func InfoPrn(im *info.ZInfoMsg) {
	fmt.Println("ztype:", im.GetZtype())
	fmt.Println("devId:", im.GetDevId())
	if im.GetDinfo() != nil {
		fmt.Println("dinfo:", im.GetDinfo())
	}
	if im.GetAinfo() != nil {
		fmt.Println("ainfo:", im.GetAinfo())
	}
	if im.GetNiinfo() != nil {
		fmt.Println("niinfo:", im.GetNiinfo())
	}
	fmt.Println("atTimeStamp:", im.GetAtTimeStamp())
	fmt.Println()
}

//ZInfoPrn print data from ZInfoMsg structure
func ZInfoPrn(im *info.ZInfoMsg, ds []*ZInfoMsgInterface, infoType ZInfoType) {
	fmt.Println("ztype:", im.GetZtype())
	fmt.Println("devId:", im.GetDevId())
	if infoType.upperType != "" {
		if infoType.lowerType != "" {
			fmt.Printf("%s.%s:\n", infoType.upperType, infoType.lowerType)
		} else {
			fmt.Printf("%s:\n", infoType.upperType)
		}
	}
	for i, d := range ds {
		fmt.Printf("[%d]: %s\n", i, *d)
	}
	fmt.Println("atTimeStamp:", im.GetAtTimeStamp())
	fmt.Println()
}

//HandleFirst runs once and interrupts the workflow of InfoWatch
func HandleFirst(im *info.ZInfoMsg, ds []*ZInfoMsgInterface, infoType ZInfoType) bool {
	//InfoPrn(im, ds)
	ZInfoPrn(im, ds, infoType)
	return true
}

//HandleAll runs for all Info's selected by InfoWatch
func HandleAll(im *info.ZInfoMsg, ds []*ZInfoMsgInterface, infoType ZInfoType) bool {
	//InfoPrn(im, ds)
	ZInfoPrn(im, ds, infoType)
	return false
}

func processElem(value reflect.Value, query map[string]string) bool {
	matched := true
	var err error
	for k, v := range query {
		// Uppercase of filed's name first letter
		var n []string
		for _, pathElement := range strings.Split(k, ".") {
			n = append(n, strings.Title(pathElement))
		}
		var clb = func(inp reflect.Value) {
			f := fmt.Sprint(inp)
			matched, err = regexp.Match(v, []byte(f))
			if err != nil {
				log.Debug(err)
			}
		}
		matched = false
		utils.LookupWithCallback(reflect.Indirect(value).Interface(), strings.Join(n, "."), clb)
		if matched == false {
			break
		}
	}
	return matched
}

//ZInfoFind finds ZInfoMsg records with 'devid' and ZInfoDevSWF structure fields
//by reqexps in 'query'
func ZInfoFind(im *info.ZInfoMsg, query map[string]string, infoType ZInfoType) []*ZInfoMsgInterface {
	var dsws []*ZInfoMsgInterface

	devid, ok := query["devId"]
	if ok {
		if devid != im.DevId {
			return nil
		}
	}

	delete(query, "devId")

	if infoType.upperType != "" {
		dInfo := reflect.ValueOf(im).MethodByName(infoType.upperType).Call([]reflect.Value{})
		if len(dInfo) != 1 || dInfo[0].Interface() == nil {
			return nil
		}
		if reflect.Indirect(reflect.ValueOf(dInfo[0].Interface())).Kind() == reflect.Invalid {
			return nil
		}
		if infoType.lowerType != "" && infoType.upperType != "" {
			dInfoField := reflect.Indirect(reflect.ValueOf(dInfo[0].Interface())).FieldByName(infoType.lowerType)
			for i := 0; i < dInfoField.Len(); i++ {
				d := dInfoField.Index(i)
				if processElem(d, query) {
					var strValT ZInfoMsgInterface = d.Interface()
					dsws = append(dsws, &strValT)
				}
			}
		} else if infoType.upperType != "" {
			d := dInfo[0]
			if processElem(d, query) {
				var strValT ZInfoMsgInterface = d.Interface()
				dsws = append(dsws, &strValT)
			}
		}
	} else {
		var d reflect.Value
		switch im.Ztype {
		case info.ZInfoTypes_ZiDevice:
			d = reflect.ValueOf(im.GetDinfo())
		case info.ZInfoTypes_ZiApp:
			d = reflect.ValueOf(im.GetAinfo())
		case info.ZInfoTypes_ZiBlobList:
			d = reflect.ValueOf(im.GetBinfo())
		case info.ZInfoTypes_ZiContentTree:
			d = reflect.ValueOf(im.GetCinfo())
		case info.ZInfoTypes_ZiVolume:
			d = reflect.ValueOf(im.GetVinfo())
		case info.ZInfoTypes_ZiNetworkInstance:
			d = reflect.ValueOf(im.GetNiinfo())
		default:
			return dsws
		}
		if processElem(d, query) {
			var strValT ZInfoMsgInterface = im
			dsws = append(dsws, &strValT)
		}
	}
	return dsws
}

//InfoFind find ZInfoMsg records by reqexps in 'query' corresponded to devId and
//ZInfoDevSW structure.
func InfoFind(im *info.ZInfoMsg, query map[string]string) int {
	matched := 1
	for k, v := range query {
		// Uppercase of filed's name first letter
		n := strings.Title(k)
		// Find field in structure by Titlized() name 'n'
		r := reflect.ValueOf(im)
		f := fmt.Sprint(reflect.Indirect(r).FieldByName(n))
		matched, err := regexp.Match(v, []byte(f))
		if err != nil {
			return -1
		}
		if matched == false {
			return 0
		}
	}
	return matched
}

//InfoCheckerMode is InfoExist, InfoNew and InfoAny
type InfoCheckerMode int

// InfoChecker modes InfoExist, InfoNew and InfoAny.
const (
	InfoExist InfoCheckerMode = iota // just look to existing files
	InfoNew                          // wait for new files
	InfoAny                          // use both mechanisms
)

func infoProcess(query map[string]string, qhandler QHandlerFunc, handler HandlerFunc, infoType ZInfoType) loaders.ProcessFunction {
	return func(bytes []byte) (bool, error) {
		im, err := ParseZInfoMsg(bytes)
		if err != nil {
			return true, nil
		}
		ds := qhandler(&im, query, infoType)
		if ds != nil {
			if handler(&im, ds, infoType) {
				return false, nil
			}
		}
		return true, nil
	}
}

//InfoLast search Info files in the 'filepath' directory according to the 'query' parameters accepted by the 'qhandler' function and subsequent process using the 'handler' function.
func InfoLast(loader loaders.Loader, query map[string]string, qhandler QHandlerFunc, handler HandlerFunc, infoType ZInfoType) error {
	return loader.ProcessExisting(infoProcess(query, qhandler, handler, infoType), loaders.InfoType)
}

//InfoWatch monitors the change of Info files in the 'filepath' directory according to the 'query' parameters accepted by the 'qhandler' function and subsequent processing using the 'handler' function with 'timeoutSeconds'.
func InfoWatch(loader loaders.Loader, query map[string]string, qhandler QHandlerFunc, handler HandlerFunc, infoType ZInfoType, timeoutSeconds time.Duration) error {
	return loader.ProcessStream(infoProcess(query, qhandler, handler, infoType), loaders.InfoType, timeoutSeconds)
}

//InfoChecker checks the information in the regular expression pattern 'query' and processes the info.ZInfoMsg found by the function 'handler' from existing files (mode=InfoExist), new files (mode=InfoNew) or any of them (mode=InfoAny) with timeout (0 for infinite).
func InfoChecker(loader loaders.Loader, devUUID uuid.UUID, query map[string]string, infoType ZInfoType, handler HandlerFunc, mode InfoCheckerMode, timeout time.Duration) (err error) {
	loader.SetUUID(devUUID)
	done := make(chan error)

	// observe new files
	if mode == InfoNew || mode == InfoAny {
		go func() {
			done <- InfoWatch(loader.Clone(), query, ZInfoFind, handler, infoType, timeout)
		}()
	}
	// check info by pattern in existing files
	if mode == InfoExist || mode == InfoAny {
		go func() {
			handler := func(im *info.ZInfoMsg, ds []*ZInfoMsgInterface, infoType ZInfoType) (result bool) {
				if result = handler(im, ds, infoType); result {
					done <- nil
				}
				return
			}
			if err = InfoLast(loader.Clone(), query, ZInfoFind, handler, infoType); err != nil {
				done <- err
			}
		}()
	}
	return <-done
}
