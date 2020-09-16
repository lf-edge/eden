//Package emetric provides primitives for searching and processing data
//in Metric files.
package emetric

import (
	"encoding/json"
	"fmt"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/ptypes"
	"github.com/lf-edge/eden/pkg/controller/loaders"
	"github.com/lf-edge/eden/pkg/controller/types"
	"github.com/lf-edge/eden/pkg/utils"
	"github.com/lf-edge/eve/api/go/metrics"
	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
	"reflect"
	"regexp"
	"strings"
	"time"
)

//MetricCheckerMode is MetricExist, MetricNew and MetricAny
type MetricCheckerMode int

//MetricTail returns MetricCheckerMode for process only defined count of last messages
func MetricTail(count uint) MetricCheckerMode {
	return MetricCheckerMode(count)
}

// MetricChecker modes MetricExist, MetricNew and MetricAny.
const (
	MetricExist MetricCheckerMode = -3 //MetricExist just look to existing files
	MetricNew   MetricCheckerMode = -2 //MetricNew wait for new files
	MetricAny   MetricCheckerMode = -1 //MetricAny use both mechanisms
)

//ParseMetricsBundle unmarshal LogBundle
func ParseMetricsBundle(data []byte) (logBundle *metrics.ZMetricMsg, err error) {
	var lb metrics.ZMetricMsg
	err = jsonpb.UnmarshalString(string(data), &lb)
	return &lb, err
}

//ParseMetricItem apply regexp on logItem
func ParseMetricItem(data string) (logItem *metrics.ZMetricMsg, err error) {
	pattern := `(?P<time>[^{]*):\s*(?P<json>{.*})`
	re := regexp.MustCompile(pattern)
	parts := re.SubexpNames()
	result := re.FindAllStringSubmatch(data, -1)
	m := map[string]string{}
	if len(result) == 0 {
		log.Debugf("error in FindAllStringSubmatch for %s and string %s. Will use new api", pattern, data)
		var le *metrics.ZMetricMsg
		err = json.Unmarshal([]byte(data), &le)
		return le, err
	}
	for i, n := range result[0] {
		m[parts[i]] = n
	}
	var le *metrics.ZMetricMsg
	err = json.Unmarshal([]byte(m["json"]), &le)

	return le, err
}

//MetricItemPrint find ZMetricMsg records by path in 'query'
func MetricItemPrint(mm *metrics.ZMetricMsg, query []string) *types.PrintResult {
	result := make(types.PrintResult)
	for _, v := range query {
		splitRequest := strings.Split(v, ".")
		if len(splitRequest) > 0 && strings.ToLower(splitRequest[0]) == "dm" { //dm is located in MetricContent
			v = strings.Join(append([]string{"MetricContent"}, splitRequest...), ".")
		}
		// Uppercase of filed's name first letter
		var n []string
		for _, pathElement := range strings.Split(v, ".") {
			n = append(n, strings.Title(pathElement))
		}
		var clb = func(inp reflect.Value) {
			f := fmt.Sprint(inp)
			result[v] = append(result[v], f)
		}
		utils.LookupWithCallback(reflect.Indirect(reflect.ValueOf(mm)).Interface(), strings.Join(n, "."), clb)
	}
	return &result
}

//MetricItemFind find ZMetricMsg records by reqexps in 'query' corresponded to ZMetricMsg structure.
func MetricItemFind(mm *metrics.ZMetricMsg, query map[string]string) bool {
	matched := true
	for k, v := range query {
		splitRequest := strings.Split(k, ".")
		if len(splitRequest) > 0 && strings.ToLower(splitRequest[0]) == "dm" { //dm is located in MetricContent
			query[strings.Join(append([]string{"MetricContent"}, splitRequest...), ".")] = v
			delete(query, k)
		}
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
		utils.LookupWithCallback(reflect.ValueOf(mm).Interface(), strings.Join(n, "."), clb)
		if matched == false {
			return matched
		}
	}
	return matched
}

//HandleFirst runs once and interrupts the workflow of LogWatch
func HandleFirst(le *metrics.ZMetricMsg) bool {
	MetricPrn(le)
	return true
}

//HandleAll runs for all Logs selected by LogWatch
func HandleAll(le *metrics.ZMetricMsg) bool {
	MetricPrn(le)
	return false
}

//MetricPrn print Metric data
func MetricPrn(le *metrics.ZMetricMsg) {
	fmt.Printf("DevID: %s", le.DevID)
	fmt.Printf("\tAtTimeStamp: %s", ptypes.TimestampString(le.AtTimeStamp))
	fmt.Print("\tDm: ", le.GetDm(), "\tAm: ", le.Am, "\tNm: ", le.Nm, "\tVm: ", le.Vm)
	fmt.Println()
}

//HandlerFunc must process ZMetricMsg and return true to exit
//or false to continue
type HandlerFunc func(msg *metrics.ZMetricMsg) bool

func metricProcess(query map[string]string, handler HandlerFunc) loaders.ProcessFunction {
	devID, ok := query["devId"]
	if ok {
		delete(query, "devId")
	}
	return func(bytes []byte) (bool, error) {
		lb, err := ParseMetricsBundle(bytes)
		if err != nil {
			return true, nil
		}
		if devID != "" && devID != lb.DevID {
			return true, nil
		}
		if MetricItemFind(lb, query) {
			if handler(lb) {
				return false, nil
			}
		}
		return true, nil
	}
}

//MetricWatch monitors the change of Metric files in the 'filepath' directory
//according to the 'query' reqexps and processing using the 'handler' function.
func MetricWatch(loader loaders.Loader, query map[string]string, handler HandlerFunc, timeoutSeconds time.Duration) error {
	return loader.ProcessStream(metricProcess(query, handler), types.MetricsType, timeoutSeconds)
}

//MetricLast function process Metric files in the 'filepath' directory
//according to the 'query' reqexps and return last founded item
func MetricLast(loader loaders.Loader, query map[string]string, handler HandlerFunc) error {
	return loader.ProcessExisting(metricProcess(query, handler), types.MetricsType)
}

//MetricChecker check metrics by pattern from existence files with HandlerFunc with timeout for observe new files
func MetricChecker(loader loaders.Loader, devUUID uuid.UUID, q map[string]string, handler HandlerFunc, mode MetricCheckerMode, timeout time.Duration) (err error) {
	loader.SetUUID(devUUID)
	done := make(chan error)

	// observe new files
	if mode == MetricNew || mode == MetricAny {
		go func() {
			done <- MetricWatch(loader.Clone(), q, handler, timeout)
		}()
	}
	// check info by pattern in existing files
	if mode == MetricExist || mode == MetricAny {
		go func() {
			handler := func(item *metrics.ZMetricMsg) (result bool) {
				if result = handler(item); result {
					done <- nil
				}
				return
			}
			done <- MetricLast(loader.Clone(), q, handler)
		}()
	}
	// use for process only defined count of last messages
	if mode > 0 {
		metricQueue := utils.InitQueueWithCapacity(int(mode))
		handlerLocal := func(item *metrics.ZMetricMsg) (result bool) {
			if err = metricQueue.Enqueue(item); err != nil {
				done <- err
			}
			return false
		}
		if err = MetricLast(loader.Clone(), q, handlerLocal); err != nil {
			done <- err
		}
		el, err := metricQueue.Dequeue()
		for err == nil {
			if result := handler(el.(*metrics.ZMetricMsg)); result {
				return nil
			}
			el, err = metricQueue.Dequeue()
		}
		return nil
	}
	return <-done
}
