// Package eflowlog provides primitives for searching and processing data
// in FlowMessage files.
package eflowlog

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/lf-edge/eden/pkg/controller/loaders"
	"github.com/lf-edge/eden/pkg/controller/types"
	"github.com/lf-edge/eden/pkg/utils"
	"github.com/lf-edge/eve-api/go/flowlog"
	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"google.golang.org/protobuf/proto"
)

// FlowLogCheckerMode is FlowLogExist, FlowLogNew and FlowLogAny
type FlowLogCheckerMode int

// FlowLogTail returns FlowLogCheckerMode for process only defined count of last messages
func FlowLogTail(count uint) FlowLogCheckerMode {
	return FlowLogCheckerMode(count)
}

// FlowLogChecker modes FlowLogExist, FlowLogNew and FlowLogAny.
const (
	FlowLogExist FlowLogCheckerMode = -3 // just look to existing files
	FlowLogNew   FlowLogCheckerMode = -2 // wait for new files
	FlowLogAny   FlowLogCheckerMode = -1 // use both mechanisms
)

// ParseFullLogEntry unmarshal FlowMessage
func ParseFullLogEntry(data []byte) (*flowlog.FlowMessage, error) {
	var lb flowlog.FlowMessage
	err := proto.Unmarshal(data, &lb)
	return &lb, err
}

// FlowLogItemPrint find FlowMessage elements by paths in 'query'
func FlowLogItemPrint(le *flowlog.FlowMessage, query []string) *types.PrintResult {
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
			result[v] = append(result[v], f)
		}
		utils.LookupWithCallback(reflect.Indirect(reflect.ValueOf(le)).Interface(), strings.Join(n, "."), clb)
	}
	return &result
}

// FlowLogItemFind find FlowMessage records by reqexps in 'query' corresponded to FlowMessage structure.
func FlowLogItemFind(le *flowlog.FlowMessage, query map[string]string) bool {
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
			newMatched, err := regexp.Match(v, []byte(f))
			if err != nil {
				log.Debug(err)
			}
			if !matched && newMatched {
				matched = newMatched
			}
		}
		matched = false
		utils.LookupWithCallback(reflect.Indirect(reflect.ValueOf(le)).Interface(), strings.Join(n, "."), clb)
		if !matched {
			return matched
		}
	}
	return matched
}

// HandleFactory implements HandlerFunc which prints log in the provided format
func HandleFactory(format types.OutputFormat, once bool) HandlerFunc {
	return func(le *flowlog.FlowMessage) bool {
		FlowLogPrn(le, format)
		return once
	}
}

// FlowLogPrn print FlowMessage data
func FlowLogPrn(le *flowlog.FlowMessage, format types.OutputFormat) {
	switch format {
	case types.OutputFormatJSON:
		enc := json.NewEncoder(os.Stdout)
		_ = enc.Encode(le)
	case types.OutputFormatLines:
		fmt.Println("devId:", le.DevId)
		fmt.Println("flows:", le.Flows)
		fmt.Println("scope:", le.Scope)
		fmt.Println("dnsReqs:", le.DnsReqs)
		fmt.Println()
	default:
		log.Errorf("unknown log format requested")
	}
}

// HandlerFunc must process FlowMessage and return true to exit
// or false to continue
type HandlerFunc func(*flowlog.FlowMessage) bool

func flowLogProcess(query map[string]string, handler HandlerFunc) loaders.ProcessFunction {
	devID, ok := query["devId"]
	if ok {
		delete(query, "devId")
	}
	_, ok = query["eveVersion"]
	if ok {
		delete(query, "eveVersion")
	}
	return func(bytes []byte) (bool, error) {
		lb, err := ParseFullLogEntry(bytes)
		if err != nil {
			return true, nil
		}
		if devID != "" && devID != lb.DevId {
			return true, nil
		}
		if FlowLogItemFind(lb, query) {
			if handler(lb) {
				return false, nil
			}
		}
		return true, nil
	}
}

// FlowLogWatch monitors the change of FlowLog files in the 'filepath' directory
// according to the 'query' reqexps and processing using the 'handler' function.
func FlowLogWatch(loader loaders.Loader, query map[string]string, handler HandlerFunc, timeoutSeconds time.Duration) error {
	return loader.ProcessStream(flowLogProcess(query, handler), types.FlowLogType, timeoutSeconds)
}

// FlowLogLast function process FlowLog files in the 'filepath' directory
// according to the 'query' reqexps and return last founded item
func FlowLogLast(loader loaders.Loader, query map[string]string, handler HandlerFunc) error {
	return loader.ProcessExisting(flowLogProcess(query, handler), types.FlowLogType)
}

// FlowLogChecker check logs by pattern from existence files with FlowLogLast and use FlowLogWatchWithTimeout with timeout for observe new files
func FlowLogChecker(loader loaders.Loader, devUUID uuid.UUID, q map[string]string, handler HandlerFunc, mode FlowLogCheckerMode, timeout time.Duration) (err error) {
	loader.SetUUID(devUUID)
	done := make(chan error)

	// observe new files
	if mode == FlowLogNew || mode == FlowLogAny {
		go func() {
			done <- FlowLogWatch(loader.Clone(), q, handler, timeout)
		}()
	}
	// check FlowLog by pattern in existing files
	if mode == FlowLogExist || mode == FlowLogAny {
		go func() {
			handler := func(item *flowlog.FlowMessage) (result bool) {
				if result = handler(item); result {
					done <- nil
				}
				return
			}
			done <- FlowLogLast(loader.Clone(), q, handler)
		}()
	}
	// use for process only defined count of last messages
	if mode > 0 {
		logQueue := utils.InitQueueWithCapacity(int(mode))
		handlerLocal := func(item *flowlog.FlowMessage) (result bool) {
			if err = logQueue.Enqueue(item); err != nil {
				done <- err
			}
			return false
		}
		if err = FlowLogLast(loader.Clone(), q, handlerLocal); err != nil {
			done <- err
		}
		el, err := logQueue.Dequeue()
		for err == nil {
			if result := handler(el.(*flowlog.FlowMessage)); result {
				return nil
			}
			el, err = logQueue.Dequeue()
		}
		return nil
	}
	return <-done
}
