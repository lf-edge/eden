//Package einfo provides primitives for searching and processing data
//in Info files.
package einfo

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/lf-edge/eden/pkg/controller/loaders"
	"github.com/lf-edge/eden/pkg/controller/types"
	"github.com/lf-edge/eden/pkg/utils"
	"github.com/lf-edge/eve/api/go/info"
	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
	"google.golang.org/protobuf/encoding/protojson"
)

//HandlerFunc must process info.ZInfoMsg and return true to exit
//or false to continue
type HandlerFunc func(im *info.ZInfoMsg, ds []*ZInfoMsgInterface) bool

//QHandlerFunc must process info.ZInfoMsg with query parameters
//and return true to exit or false to continue
type QHandlerFunc func(im *info.ZInfoMsg, query map[string]string) []*ZInfoMsgInterface

//ZInfoMsgInterface is an interface to pass between handlers
type ZInfoMsgInterface interface{}

//ParseZInfoMsg unmarshal ZInfoMsg
func ParseZInfoMsg(data []byte) (ZInfoMsg *info.ZInfoMsg, err error) {
	var zi info.ZInfoMsg
	err = protojson.Unmarshal(data, &zi)
	return &zi, err
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
	fmt.Println("atTimeStamp:", im.GetAtTimeStamp().AsTime())
	fmt.Println()
}

//ZInfoPrn print data from ZInfoMsg structure
func ZInfoPrn(im *info.ZInfoMsg, ds []*ZInfoMsgInterface) {
	fmt.Println("ztype:", im.GetZtype())
	fmt.Println("devId:", im.GetDevId())
	for i, d := range ds {
		fmt.Printf("[%d]: %s\n", i, *d)
	}
	fmt.Println("atTimeStamp:", im.GetAtTimeStamp().AsTime())
	fmt.Println()
}

//HandleFirst runs once and interrupts the workflow of InfoWatch
func HandleFirst(im *info.ZInfoMsg, ds []*ZInfoMsgInterface) bool {
	//InfoPrn(im, ds)
	ZInfoPrn(im, ds)
	return true
}

//HandleAll runs for all Info's selected by InfoWatch
func HandleAll(im *info.ZInfoMsg, ds []*ZInfoMsgInterface) bool {
	//InfoPrn(im, ds)
	ZInfoPrn(im, ds)
	return false
}

func processElem(value reflect.Value, query map[string]string) bool {
	matched := true
	for k, v := range query {
		// Uppercase of filed's name first letter
		var n []string
		for _, pathElement := range strings.Split(k, ".") {
			n = append(n, strings.Title(pathElement))
		}
		var clb = func(inp reflect.Value) {
			f := fmt.Sprintf("%s", inp)
			newMatched, err := regexp.Match(v, []byte(f))
			if err != nil {
				log.Debug(err)
			}
			if !matched && newMatched {
				matched = newMatched
			}
		}
		matched = false
		utils.LookupWithCallback(reflect.Indirect(value).Interface(), strings.Join(n, "."), clb)
		if !matched {
			break
		}
	}
	return matched
}

//ZInfoPrint finds ZInfoMsg records by path in 'query'
func ZInfoPrint(im *info.ZInfoMsg, query []string) *types.PrintResult {
	result := make(types.PrintResult)
	for _, v := range query {
		// Uppercase of filed's name first letter
		var n []string
		for _, pathElement := range strings.Split(v, ".") {
			n = append(n, strings.Title(pathElement))
		}
		var clb = func(inp reflect.Value) {
			f := fmt.Sprintf("%s", inp)
			result[v] = append(result[v], f)
		}
		utils.LookupWithCallback(reflect.Indirect(reflect.ValueOf(im)).Interface(), strings.Join(n, "."), clb)
	}
	return &result
}

//ZInfoFind finds ZInfoMsg records with 'devid' and ZInfoDevSWF structure fields
//by reqexps in 'query'
func ZInfoFind(im *info.ZInfoMsg, query map[string]string) []*ZInfoMsgInterface {
	var dsws []*ZInfoMsgInterface
	if processElem(reflect.ValueOf(im), query) {
		var strValT ZInfoMsgInterface = im
		dsws = append(dsws, &strValT)
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
		if !matched {
			return 0
		}
	}
	return matched
}

//InfoCheckerMode is InfoExist, InfoNew and InfoAny
type InfoCheckerMode int

//InfoTail returns InfoCheckerMode for process only defined count of last messages
func InfoTail(count uint) InfoCheckerMode {
	return InfoCheckerMode(count)
}

// InfoChecker modes InfoExist, InfoNew and InfoAny.
const (
	InfoExist InfoCheckerMode = -3 // just look to existing files
	InfoNew   InfoCheckerMode = -2 // wait for new files
	InfoAny   InfoCheckerMode = -1 // use both mechanisms
)

func infoProcess(query map[string]string, qhandler QHandlerFunc, handler HandlerFunc) loaders.ProcessFunction {
	return func(bytes []byte) (bool, error) {
		im, err := ParseZInfoMsg(bytes)
		if err != nil {
			return true, nil
		}
		ds := qhandler(im, query)
		if ds != nil {
			if handler(im, ds) {
				return false, nil
			}
		}
		return true, nil
	}
}

//InfoLast search Info files in the 'filepath' directory according to the 'query' parameters accepted by the 'qhandler' function and subsequent process using the 'handler' function.
func InfoLast(loader loaders.Loader, query map[string]string, qhandler QHandlerFunc, handler HandlerFunc) error {
	return loader.ProcessExisting(infoProcess(query, qhandler, handler), types.InfoType)
}

//InfoWatch monitors the change of Info files in the 'filepath' directory according to the 'query' parameters accepted by the 'qhandler' function and subsequent processing using the 'handler' function with 'timeoutSeconds'.
func InfoWatch(loader loaders.Loader, query map[string]string, qhandler QHandlerFunc, handler HandlerFunc, timeoutSeconds time.Duration) error {
	return loader.ProcessStream(infoProcess(query, qhandler, handler), types.InfoType, timeoutSeconds)
}

//InfoChecker checks the information in the regular expression pattern 'query' and processes the info.ZInfoMsg found by the function 'handler' from existing files (mode=InfoExist), new files (mode=InfoNew) or any of them (mode=InfoAny) with timeout (0 for infinite).
func InfoChecker(loader loaders.Loader, devUUID uuid.UUID, query map[string]string, handler HandlerFunc, mode InfoCheckerMode, timeout time.Duration) (err error) {
	loader.SetUUID(devUUID)
	done := make(chan error)

	// observe new files
	if mode == InfoNew || mode == InfoAny {
		go func() {
			done <- InfoWatch(loader.Clone(), query, ZInfoFind, handler, timeout)
		}()
	}
	// check info by pattern in existing files
	if mode == InfoExist || mode == InfoAny {
		go func() {
			handler := func(im *info.ZInfoMsg, ds []*ZInfoMsgInterface) (result bool) {
				if result = handler(im, ds); result {
					done <- nil
				}
				return
			}
			done <- InfoLast(loader.Clone(), query, ZInfoFind, handler)
		}()
	}
	// use for process only defined count of last messages
	if mode > 0 {
		type infoSave struct {
			im *info.ZInfoMsg
			ds []*ZInfoMsgInterface
		}
		infoQueue := utils.InitQueueWithCapacity(int(mode))
		handlerLocal := func(im *info.ZInfoMsg, ds []*ZInfoMsgInterface) (result bool) {
			if err = infoQueue.Enqueue(infoSave{im: im, ds: ds}); err != nil {
				done <- err
			}
			return false
		}
		if err = InfoLast(loader.Clone(), query, ZInfoFind, handlerLocal); err != nil {
			done <- err
		}
		el, err := infoQueue.Dequeue()
		for err == nil {
			if result := handler(el.(infoSave).im, el.(infoSave).ds); result {
				return nil
			}
			el, err = infoQueue.Dequeue()
		}
		return nil
	}
	return <-done
}
