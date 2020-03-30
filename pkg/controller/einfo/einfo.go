//Package einfo provides primitives for searching and processing data
//in Info files.
package einfo

import (
	"errors"
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/golang/protobuf/jsonpb"
	"github.com/lf-edge/eve/api/go/info"
	"io/ioutil"
	"log"
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
	//ZInfoDevSW can be used for filter GetDinfo SwList
	ZInfoDevSW ZInfoType = &zInfoPacket{upperType: "GetDinfo", lowerType: "SwList"}
	//ZInfoNetwork can be used for filter GetDinfo Network
	ZInfoNetwork ZInfoType = &zInfoPacket{upperType: "GetDinfo", lowerType: "Network"}
	//ZInfoNetworkInstance can be used for filter GetNiinfo
	ZInfoNetworkInstance ZInfoType = &zInfoPacket{upperType: "GetNiinfo"}
)

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
	fmt.Printf("%s.%s:\n", infoType.upperType, infoType.lowerType)
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

	dInfo := reflect.ValueOf(im).MethodByName(infoType.upperType).Call([]reflect.Value{})
	if len(dInfo) != 1 || dInfo[0].Interface() == nil {
		return nil
	}
	if reflect.Indirect(reflect.ValueOf(dInfo[0].Interface())).Kind() == reflect.Invalid {
		return nil
	}
	if infoType.lowerType != "" {
		dInfoField := reflect.Indirect(reflect.ValueOf(dInfo[0].Interface())).FieldByName(infoType.lowerType)
		for i := 0; i < dInfoField.Len(); i++ {
			d := dInfoField.Index(i)
			if processElem(d, query) {
				var strValT ZInfoMsgInterface = d.Interface()
				dsws = append(dsws, &strValT)
			}
		}
	} else {
		d := dInfo[0]
		if processElem(d, query) {
			var strValT ZInfoMsgInterface = d.Interface()
			dsws = append(dsws, &strValT)
		}
	}
	return dsws
}

//InfoWatchWithTimeout monitors the change of Info files in the 'filepath' directory with 'timeoutSeconds' according to the 'query' parameters accepted by the 'qhandler' function and subsequent processing using the 'handler' function.
func InfoWatchWithTimeout(filepath string, query map[string]string, qhandler QHandlerFunc, handler HandlerFunc, infoType ZInfoType, timeoutSeconds time.Duration) error {
	done := make(chan error)
	go func() {
		err := InfoWatch(filepath, query, qhandler, handler, infoType)
		if err != nil {
			done <- err
			return
		}
		done <- nil
	}()
	select {
	case err := <-done:
		return err
	case <-time.After(timeoutSeconds * time.Second):
		return errors.New("timeout")
	}
}

//InfoWatch monitors the change of Info files in the 'filepath' directory according to the 'query' parameters accepted by the 'qhandler' function and subsequent processing using the 'handler' function.
func InfoWatch(filepath string, query map[string]string, qhandler QHandlerFunc, handler HandlerFunc, infoType ZInfoType) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()

	done := make(chan bool)
	go func() {
		defer func() { done <- true }()
		for {
			select {
			case event := <-watcher.Events:
				switch event.Op {
				case fsnotify.Write:
					time.Sleep(1 * time.Second) // wait for write ends
					data, err := ioutil.ReadFile(event.Name)
					if err != nil {
						log.Print("Can't open", event.Name)
						continue
					}

					im, err := ParseZInfoMsg(data)
					if err != nil {
						log.Print("Can't parse ZInfoMsg", event.Name)
						continue
					}
					ds := qhandler(&im, query, infoType)
					if ds != nil {
						if handler(&im, ds, infoType) {
							return
						}
					}

					continue
				}
			case err := <-watcher.Errors:
				log.Printf("Error: %s", err)
			}
		}
	}()

	err = watcher.Add(filepath)
	if err != nil {
		return err
	}

	<-done
	return nil
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
		data, err := ioutil.ReadFile(fileFullPath)
		if err != nil {
			log.Print("Can't open ", fileFullPath)
			continue
		}

		im, err := ParseZInfoMsg(data)
		if err != nil {
			log.Print("Can't parse ZInfoMsg ", fileFullPath)
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

//InfoChecker check info by pattern from existence files with InfoLast and use InfoWatchWithTimeout with timeout for observe new files
func InfoChecker(dir string, q map[string]string, infoType ZInfoType, timeout time.Duration) (err error) {
	done := make(chan error)

	go func() {
		err = InfoWatchWithTimeout(dir, q, ZInfoFind, HandleFirst, infoType, timeout)
		done <- err
	}()
	go func() {
		handler := func(im *info.ZInfoMsg, ds []*ZInfoMsgInterface, infoType ZInfoType) bool {
			ZInfoPrn(im, ds, infoType)
			done <- nil
			return true
		}
		err = InfoLast(dir, q, ZInfoFind, handler, infoType)
		if err != nil {
			done <- err
		}
	}()
	return <-done
}
