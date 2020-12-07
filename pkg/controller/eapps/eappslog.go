//Package eapps provides primitives for searching and processing data
//in Log files of apps.
package eapps

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
	"github.com/lf-edge/eve/api/go/logs"
	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
	"google.golang.org/protobuf/encoding/protojson"
)

//LogCheckerMode is InfoExist, InfoNew and InfoAny
type LogCheckerMode int

// LogFormat the format to print output logs
type LogFormat byte

const (
	//LogLines returns log line by line
	LogLines LogFormat = iota
	//LogJSON returns log in JSON format
	LogJSON
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

//ParseLogEntry unmarshal LogEntry
func ParseLogEntry(data []byte) (logEntry *logs.LogEntry, err error) {
	var le logs.LogEntry
	err = protojson.Unmarshal(data, &le)
	return &le, err
}

//LogItemFind find LogItem records by reqexps in 'query' corresponded to LogItem structure.
func LogItemFind(le *logs.LogEntry, query map[string]string) bool {
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
		utils.LookupWithCallback(reflect.Indirect(reflect.ValueOf(le)).Interface(), strings.Join(n, "."), clb)
		if !matched {
			return matched
		}
	}
	return matched
}

//HandleFactory implements HandlerFunc which prints log in the provided format
func HandleFactory(format LogFormat, once bool) HandlerFunc {
	return func(le *logs.LogEntry) bool {
		LogPrn(le, format)
		return once
	}
}

//LogPrn print Log data
func LogPrn(le *logs.LogEntry, format LogFormat) {
	switch format {
	case LogJSON:
		enc := json.NewEncoder(os.Stdout)
		_ = enc.Encode(le)
	case LogLines:
		fmt.Println("source:", le.Source)
		fmt.Println("severity:", le.Severity)
		fmt.Println("content:", strings.TrimSpace(le.Content))
		fmt.Println("filename:", le.Filename)
		fmt.Println("function:", le.Function)
		fmt.Println("timestamp:", le.Timestamp)
		fmt.Println("iid:", le.Iid)
		fmt.Println()
	default:
		_, _ = fmt.Fprintf(os.Stderr, "unknown log format requested")
	}
}

//HandlerFunc must process LogItem and return true to exit
//or false to continue
type HandlerFunc func(*logs.LogEntry) bool

func logProcess(query map[string]string, handler HandlerFunc) loaders.ProcessFunction {
	return func(bytes []byte) (bool, error) {
		le, err := ParseLogEntry(bytes)
		if err != nil {
			return true, nil
		}
		if LogItemFind(le, query) {
			if handler(le) {
				return false, nil
			}
		}
		return true, nil
	}
}

//LogWatch monitors the change of Log files in the 'filepath' directory
//according to the 'query' reqexps and processing using the 'handler' function.
func LogWatch(loader loaders.Loader, query map[string]string, handler HandlerFunc, timeoutSeconds time.Duration) error {
	return loader.ProcessStream(logProcess(query, handler), types.AppsType, timeoutSeconds)
}

//LogLast function process Log files in the 'filepath' directory
//according to the 'query' reqexps and return last founded item
func LogLast(loader loaders.Loader, query map[string]string, handler HandlerFunc) error {
	return loader.ProcessExisting(logProcess(query, handler), types.AppsType)
}

//LogChecker check logs by pattern from existence files with LogLast and use LogWatchWithTimeout with timeout for observe new files
func LogChecker(loader loaders.Loader, devUUID uuid.UUID, appUUID uuid.UUID, q map[string]string, handler HandlerFunc, mode LogCheckerMode, timeout time.Duration) (err error) {
	loader.SetUUID(devUUID)
	loader.SetAppUUID(appUUID)
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
			handler := func(item *logs.LogEntry) (result bool) {
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
		handlerLocal := func(item *logs.LogEntry) (result bool) {
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
			if result := handler(el.(*logs.LogEntry)); result {
				return nil
			}
			el, err = logQueue.Dequeue()
		}
		return nil
	}
	return <-done
}
