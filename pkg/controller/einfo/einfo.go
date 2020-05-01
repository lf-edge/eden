//Package einfo provides primitives for searching and processing data
//in Info files.
package einfo

import (
	"errors"
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/golang/protobuf/jsonpb"
	"github.com/lf-edge/eve/api/go/info"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"path"
	"reflect"
	"regexp"
	"sort"
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
		n := strings.Title(k)
		f := fmt.Sprint(reflect.Indirect(value).FieldByName(n))
		matched, err = regexp.Match(v, []byte(f))
		if err != nil {
			log.Print(err)
			return false
		}
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
		var strValT ZInfoMsgInterface = im
		dsws = append(dsws, &strValT)
	}
	return dsws
}

//InfoWatch monitors the change of Info files in the 'filepath' directory according to the 'query' parameters accepted by the 'qhandler' function and subsequent processing using the 'handler' function with 'timeoutSeconds'.
func InfoWatch(filepath string, query map[string]string, qhandler QHandlerFunc, handler HandlerFunc, infoType ZInfoType, timeoutSeconds time.Duration) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	if timeoutSeconds == 0 {
		timeoutSeconds = -1
	}

	done := make(chan error)
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					done <- errors.New("watcher closed")
					return
				}
				switch event.Op {
				case fsnotify.Write:
					time.Sleep(1 * time.Second) // wait for write ends
					data, err := ioutil.ReadFile(event.Name)
					if err != nil {
						log.Error("Can't open", event.Name)
						continue
					}
					log.Debugf("parse info file %s", event.Name)

					im, err := ParseZInfoMsg(data)
					if err != nil {
						log.Error("Can't parse ZInfoMsg", event.Name)
						continue
					}
					ds := qhandler(&im, query, infoType)
					if ds != nil {
						if handler(&im, ds, infoType) {
							done <- nil
							return
						}
					}

					continue
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					done <- err
					return
				}
				log.Errorf("error: %s", err)
			case <-time.After(timeoutSeconds * time.Second):
				done <- errors.New("timeout")
				return
			}
		}
	}()

	err = watcher.Add(filepath)
	if err != nil {
		return err
	}

	err = <-done
	_ = watcher.Close()
	return err
}

//InfoLast search Info files in the 'filepath' directory according to the 'query' parameters accepted by the 'qhandler' function and subsequent process using the 'handler' function.
func InfoLast(filepath string, query map[string]string, qhandler QHandlerFunc, handler HandlerFunc, infoType ZInfoType) error {
	files, err := ioutil.ReadDir(filepath)
	if err != nil {
		return err
	}
	sort.Slice(files, func(i, j int) bool {
		return files[i].ModTime().Unix() > files[j].ModTime().Unix()
	})
	time.Sleep(1 * time.Second) // wait for write ends
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		fileFullPath := path.Join(filepath, file.Name())
		log.Debugf("parse info file %s", fileFullPath)
		data, err := ioutil.ReadFile(fileFullPath)
		if err != nil {
			log.Error("Can't open ", fileFullPath)
			continue
		}

		im, err := ParseZInfoMsg(data)
		if err != nil {
			log.Error("Can't parse ZInfoMsg ", fileFullPath)
			continue
		}
		ds := qhandler(&im, query, infoType)
		if ds != nil {
			if handler(&im, ds, infoType) {
				return nil
			}
		}
		continue
	}
	return nil
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

//InfoChecker checks the information in the regular expression pattern 'query' and processes the info.ZInfoMsg found by the function 'handler' from existing files (mode=InfoExist), new files (mode=InfoNew) or any of them (mode=InfoAny) with timeout (0 for infinite).
func InfoChecker(dir string, query map[string]string, infoType ZInfoType, handler HandlerFunc, mode InfoCheckerMode, timeout time.Duration) (err error) {
	log.Infof("wait for info in %s", dir)
	done := make(chan error)

	// observe new files
	if mode == InfoNew || mode == InfoAny {
		go func() {
			err = InfoWatch(dir, query, ZInfoFind, handler, infoType, timeout)
			done <- err
		}()
	}
	// check info by pattern in existing files
	if mode == InfoExist || mode == InfoAny {
		go func() {
			handler := func(im *info.ZInfoMsg, ds []*ZInfoMsgInterface, infoType ZInfoType) bool {
				handler(im, ds, infoType)
				done <- nil
				return true
			}
			err = InfoLast(dir, query, ZInfoFind, handler, infoType)
			if err != nil {
				done <- err
			}
		}()
	}
	return <-done
}
