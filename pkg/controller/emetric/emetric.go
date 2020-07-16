//Package emetric provides primitives for searching and processing data
//in Metric files.
package emetric

import (
	"encoding/json"
	"fmt"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/ptypes"
	"github.com/lf-edge/eden/pkg/controller/loaders"
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

const (
	MetricExist MetricCheckerMode = iota // just look to existing files
	MetricNew                            // wait for new files
	MetricAny                            // use both mechanisms
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

//MetricItemFind find LogItem records by reqexps in 'query' corresponded to LogItem structure.
func MetricItemFind(le *metrics.ZMetricMsg, query map[string]string) bool {
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
		utils.LookupWithCallback(reflect.ValueOf(le).Interface(), strings.Join(n, "."), clb)
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
		for el, i := range query {
			splitRequest := strings.Split(el, ".")
			if len(splitRequest) > 0 && strings.ToLower(splitRequest[0]) == "dm" { //dm is located in MetricContent
				query[strings.Join(append([]string{"MetricContent"}, splitRequest...), ".")] = i
				delete(query, el)
			}
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
	return loader.ProcessStream(metricProcess(query, handler), loaders.MetricsType, timeoutSeconds)
}

//MetricLast function process Metric files in the 'filepath' directory
//according to the 'query' reqexps and return last founded item
func MetricLast(loader loaders.Loader, query map[string]string, handler HandlerFunc) error {
	return loader.ProcessExisting(metricProcess(query, handler), loaders.MetricsType)
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
			err := MetricLast(loader.Clone(), q, handler)
			if err != nil {
				done <- err
			}
		}()
	}
	return <-done
}
