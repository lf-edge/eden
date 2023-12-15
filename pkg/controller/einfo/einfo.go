// Package einfo provides primitives for searching and processing data
// in Info files.
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
	"github.com/lf-edge/eve-api/go/info"
	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// HandlerFunc must process info.ZInfoMsg and return true to exit
// or false to continue
type HandlerFunc func(im *info.ZInfoMsg) bool

// QHandlerFunc must process info.ZInfoMsg with query parameters
// and return true to exit or false to continue
type QHandlerFunc func(im *info.ZInfoMsg, query map[string]string) bool

// ParseZInfoMsg unmarshal ZInfoMsg
func ParseZInfoMsg(data []byte) (ZInfoMsg *info.ZInfoMsg, err error) {
	var zi info.ZInfoMsg
	err = proto.Unmarshal(data, &zi)
	return &zi, err
}

// ZInfoPrn print data from ZInfoMsg structure
func ZInfoPrn(im *info.ZInfoMsg, format types.OutputFormat) {
	switch format {
	case types.OutputFormatJSON:
		b, err := protojson.Marshal(im)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(string(b))
	case types.OutputFormatLines:
		fmt.Println("ztype:", im.GetZtype())
		fmt.Println("devId:", im.GetDevId())
		fmt.Println("content:", im)
		fmt.Println("atTimeStamp:", im.GetAtTimeStamp().AsTime())
		fmt.Println()
	default:
		log.Errorf("unknown log format requested")
	}
}

// HandleFactory implements HandlerFunc which prints info in the provided format
func HandleFactory(format types.OutputFormat, once bool) HandlerFunc {
	return func(le *info.ZInfoMsg) bool {
		ZInfoPrn(le, format)
		return once
	}
}

func processElem(value reflect.Value, query map[string]string) bool {
	matched := true
	for k, v := range query {
		// Uppercase of filed's name first letter
		var n []string
		caser := cases.Title(language.English, cases.NoLower)
		for _, pathElement := range strings.Split(k, ".") {
			n = append(n, caser.String(pathElement))
		}
		var clb = func(inp reflect.Value) {
			f := fmt.Sprint(inp)
			if b, ok := inp.Interface().([]byte); ok {
				f = fmt.Sprintf("%s", b)
			}
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

// ZInfoPrintFiltered finds ZInfoMsg records by path in 'query'
func ZInfoPrintFiltered(im *info.ZInfoMsg, query []string) *types.PrintResult {
	result := make(types.PrintResult)
	for _, v := range query {
		// Uppercase of filed's name first letter
		var n []string
		caser := cases.Title(language.English, cases.NoLower)
		for _, pathElement := range strings.Split(v, ".") {
			n = append(n, caser.String(pathElement))
		}
		var clb = func(inp reflect.Value) {
			f := fmt.Sprint(inp)
			if b, ok := inp.Interface().([]byte); ok {
				f = fmt.Sprintf("%s", b)
			}
			result[v] = append(result[v], f)
		}
		utils.LookupWithCallback(reflect.Indirect(reflect.ValueOf(im)).Interface(), strings.Join(n, "."), clb)
	}
	return &result
}

// ZInfoFind finds ZInfoMsg records with 'devid' and ZInfoDevSWF structure fields
// by reqexps in 'query'
func ZInfoFind(im *info.ZInfoMsg, query map[string]string) bool {
	if processElem(reflect.ValueOf(im), query) {
		return true
	}
	return false
}

// InfoCheckerMode is InfoExist, InfoNew and InfoAny
type InfoCheckerMode int

// InfoTail returns InfoCheckerMode for process only defined count of last messages
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
		if qhandler(im, query) {
			if handler(im) {
				return false, nil
			}
		}
		return true, nil
	}
}

// InfoLast search Info files in the 'filepath' directory according to the 'query' parameters accepted by the 'qhandler' function and subsequent process using the 'handler' function.
func InfoLast(loader loaders.Loader, query map[string]string, qhandler QHandlerFunc, handler HandlerFunc) error {
	return loader.ProcessExisting(infoProcess(query, qhandler, handler), types.InfoType)
}

// InfoWatch monitors the change of Info files in the 'filepath' directory according to the 'query' parameters accepted by the 'qhandler' function and subsequent processing using the 'handler' function with 'timeoutSeconds'.
func InfoWatch(loader loaders.Loader, query map[string]string, qhandler QHandlerFunc, handler HandlerFunc, timeoutSeconds time.Duration) error {
	return loader.ProcessStream(infoProcess(query, qhandler, handler), types.InfoType, timeoutSeconds)
}

// InfoChecker checks the information in the regular expression pattern 'query' and processes the info.ZInfoMsg found by the function 'handler' from existing files (mode=InfoExist), new files (mode=InfoNew) or any of them (mode=InfoAny) with timeout (0 for infinite).
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
			handler := func(im *info.ZInfoMsg) (result bool) {
				if handler(im) {
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
		}
		infoQueue := utils.InitQueueWithCapacity(int(mode))
		handlerLocal := func(im *info.ZInfoMsg) (result bool) {
			if err = infoQueue.Enqueue(infoSave{im: im}); err != nil {
				done <- err
			}
			return false
		}
		if err = InfoLast(loader.Clone(), query, ZInfoFind, handlerLocal); err != nil {
			done <- err
		}
		el, err := infoQueue.Dequeue()
		for err == nil {
			if result := handler(el.(infoSave).im); result {
				return nil
			}
			el, err = infoQueue.Dequeue()
		}
		return nil
	}
	return <-done
}
