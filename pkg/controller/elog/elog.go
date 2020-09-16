//Package elog provides primitives for searching and processing data
//in Log files.
package elog

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/golang/protobuf/jsonpb"
	"github.com/lf-edge/eden/pkg/controller/loaders"
	"github.com/lf-edge/eden/pkg/controller/types"
	"github.com/lf-edge/eden/pkg/utils"
	"github.com/lf-edge/eve/api/go/logs"
	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
)

//LogItem is the structure for saving log fields
type LogItem struct {
	Source    string      `json:"source"`
	Level     string      `json:"level"`
	Msg       string      `json:"msg"`
	File      string      `json:"file"`
	Func      string      `json:"func"`
	Time      string      `json:"time"`
	Pid       interface{} `json:"pid"`
	Partition string      `json:"partition"`
}

//LogCheckerMode is InfoExist, InfoNew and InfoAny
type LogCheckerMode int

// LogFormat the format to print output logs
type LogFormat byte

const (
	LogLines LogFormat = iota
	LogJson
)

//LogTail returns LogCheckerMode for process only defined count of last messages
func LogTail(count uint) LogCheckerMode {
	return LogCheckerMode(count)
}

// LogChecker modes LogExist, LogNew and LogAny.
const (
	LogExist LogCheckerMode = -3 // just look to existing files
	LogNew   LogCheckerMode = -2 // wait for new files
	LogAny   LogCheckerMode = -1 // use both mechanisms
)

//ParseLogBundle unmarshal LogBundle
func ParseLogBundle(data []byte) (logBundle logs.LogBundle, err error) {
	var lb logs.LogBundle
	err = jsonpb.UnmarshalString(string(data), &lb)
	return lb, err
}

//ParseLogItem apply regexp on logItem
func ParseLogItem(data string) (logItem LogItem, err error) {
	pattern := `(?P<time>[^{]*):\s*(?P<json>{.*})`
	re := regexp.MustCompile(pattern)
	parts := re.SubexpNames()
	result := re.FindAllStringSubmatch(data, -1)
	m := map[string]string{}
	if len(result) == 0 {
		log.Debugf("error in FindAllStringSubmatch for %s and string %s. Will use new api", pattern, data)
		var le LogItem
		err = json.Unmarshal([]byte(data), &le)
		return le, err
	}
	for i, n := range result[0] {
		m[parts[i]] = n
	}
	var le LogItem
	err = json.Unmarshal([]byte(m["json"]), &le)

	return le, err
}

//LogItemPrint find LogItem elements by paths in 'query'
func LogItemPrint(le *LogItem, format LogFormat, query []string) *types.PrintResult {
	result := make(types.PrintResult)
	for _, v := range query {
		// Uppercase of filed's name first letter
		var n []string
		for _, pathElement := range strings.Split(v, ".") {
			n = append(n, strings.Title(pathElement))
		}
		var clb = func(inp reflect.Value) {
			f := fmt.Sprint(inp)
			result[v] = append(result[v], f)
		}
		utils.LookupWithCallback(reflect.Indirect(reflect.ValueOf(le)).Interface(), strings.Join(n, "."), clb)
	}
	return &result
}

//LogItemFind find LogItem records by reqexps in 'query' corresponded to LogItem structure.
func LogItemFind(le LogItem, query map[string]string) bool {
	matched := true
	for k, v := range query {
		// Uppercase of filed's name first letter
		var n []string
		for _, pathElement := range strings.Split(k, ".") {
			n = append(n, strings.Title(pathElement))
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
		utils.LookupWithCallback(reflect.ValueOf(le).Interface(), strings.Join(n, "."), clb)
		if matched == false {
			return matched
		}
	}
	return matched
}

func HandleFactory(format LogFormat, once bool) HandlerFunc {
	return func(le *LogItem) bool {
		LogPrn(le, format)
		return once
	}
}

//LogPrn print Log data
func LogPrn(le *LogItem, format LogFormat) {
	switch format {
	case LogJson:
		enc := json.NewEncoder(os.Stdout)
		enc.Encode(le)
	case LogLines:
		fmt.Println("source:", le.Source)
		fmt.Println("level:", le.Level)
		fmt.Println("msg:", le.Msg)
		fmt.Println("file:", le.File)
		fmt.Println("func:", le.Func)
		fmt.Println("time:", le.Time)
		fmt.Println("pid:", le.Pid)
		fmt.Println("partition:", le.Partition)
		fmt.Println()
	default:
		fmt.Fprintf(os.Stderr, "unknown log format requested")
	}
}

//HandlerFunc must process LogItem and return true to exit
//or false to continue
type HandlerFunc func(*LogItem) bool

func logProcess(query map[string]string, handler HandlerFunc) loaders.ProcessFunction {
	devID, ok := query["devId"]
	if ok {
		delete(query, "devId")
	}
	eveVersion, ok := query["eveVersion"]
	if ok {
		delete(query, "eveVersion")
	}
	return func(bytes []byte) (bool, error) {
		lb, err := ParseLogBundle(bytes)
		if err != nil {
			return true, nil
		}
		if devID != "" && devID != lb.DevID {
			return true, nil
		}
		if eveVersion != "" && eveVersion != lb.EveVersion {
			return true, nil
		}
		for _, n := range lb.Log {
			s := n.Content
			le, err := ParseLogItem(s)
			if err != nil {
				log.Debugf("logProcess: %s", err)
				continue
			}
			if LogItemFind(le, query) {
				if handler(&le) {
					return false, nil
				}
			}
		}
		return true, nil
	}
}

//LogWatch monitors the change of Log files in the 'filepath' directory
//according to the 'query' reqexps and processing using the 'handler' function.
func LogWatch(loader loaders.Loader, query map[string]string, handler HandlerFunc, timeoutSeconds time.Duration) error {
	return loader.ProcessStream(logProcess(query, handler), types.LogsType, timeoutSeconds)
}

//LogLast function process Log files in the 'filepath' directory
//according to the 'query' reqexps and return last founded item
func LogLast(loader loaders.Loader, query map[string]string, handler HandlerFunc) error {
	return loader.ProcessExisting(logProcess(query, handler), types.LogsType)
}

//LogChecker check logs by pattern from existence files with LogLast and use LogWatchWithTimeout with timeout for observe new files
func LogChecker(loader loaders.Loader, devUUID uuid.UUID, q map[string]string, handler HandlerFunc, mode LogCheckerMode, timeout time.Duration) (err error) {
	loader.SetUUID(devUUID)
	done := make(chan error)

	// observe new files
	if mode == LogNew || mode == LogAny {
		go func() {
			done <- LogWatch(loader.Clone(), q, handler, timeout)
		}()
	}
	// check info by pattern in existing files
	if mode == LogExist || mode == LogAny {
		go func() {
			handler := func(item *LogItem) (result bool) {
				if result = handler(item); result {
					done <- nil
				}
				return
			}
			done <- LogLast(loader.Clone(), q, handler)
		}()
	}
	// use for process only defined count of last messages
	if mode > 0 {
		logQueue := utils.InitQueueWithCapacity(int(mode))
		handlerLocal := func(item *LogItem) (result bool) {
			if err = logQueue.Enqueue(item); err != nil {
				done <- err
			}
			return false
		}
		if err = LogLast(loader.Clone(), q, handlerLocal); err != nil {
			done <- err
		}
		el, err := logQueue.Dequeue()
		for err == nil {
			if result := handler(el.(*LogItem)); result {
				return nil
			}
			el, err = logQueue.Dequeue()
		}
		return nil
	}
	return <-done
}
