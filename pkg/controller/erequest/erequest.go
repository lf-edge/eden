//Package erequest provides primitives for searching and processing data
//in Request files.
package erequest

import (
	"encoding/json"
	"fmt"
	"github.com/lf-edge/adam/pkg/driver/common"
	"github.com/lf-edge/eden/pkg/controller/loaders"
	"github.com/lf-edge/eden/pkg/controller/types"
	"github.com/lf-edge/eden/pkg/utils"
	log "github.com/sirupsen/logrus"
	"os"
	"reflect"
	"regexp"
	"strings"
)

// RequestFormat the format to print output Requests
type RequestFormat byte

const (
	//RequestLines returns requests line by line
	RequestLines RequestFormat = iota
	//RequestJSON returns requests in JSON format
	RequestJSON
)

//ParseRequestItem apply regexp on ApiRequest
func ParseRequestItem(data []byte) (logItem common.ApiRequest, err error) {
	var le common.ApiRequest
	err = json.Unmarshal(data, &le)
	return le, err
}

//RequestItemFind find ApiRequest records by reqexps in 'query' corresponded to ApiRequest structure.
func RequestItemFind(le common.ApiRequest, query map[string]string) bool {
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

//RequestPrn print ApiRequest data
func RequestPrn(le *common.ApiRequest, format RequestFormat) {
	switch format {
	case RequestJSON:
		enc := json.NewEncoder(os.Stdout)
		enc.Encode(le)
	case RequestLines:
		fmt.Println("uuid:", le.UUID)
		fmt.Println("client-ip:", le.ClientIP)
		fmt.Println("forwarded:", le.Forwarded)
		fmt.Println("method:", le.Method)
		fmt.Println("url:", le.URL)
		fmt.Println("timestamp:", le.Timestamp)
		fmt.Println()
	default:
		fmt.Fprintf(os.Stderr, "unknown RequestFormat requested")
	}
}

//HandlerFunc must process ApiRequest and return true to exit
//or false to continue
type HandlerFunc func(request *common.ApiRequest) bool

func requestProcess(query map[string]string, handler HandlerFunc) loaders.ProcessFunction {
	return func(bytes []byte) (bool, error) {
		le, err := ParseRequestItem(bytes)
		if err != nil {
			log.Debugf("logProcess: %s", err)
		}
		if RequestItemFind(le, query) {
			if handler(&le) {
				return false, nil
			}
		}
		return true, nil
	}
}

//RequestLast function process Request files in the 'filepath' directory
//according to the 'query' reqexps and return last founded item
func RequestLast(loader loaders.Loader, query map[string]string, handler HandlerFunc) error {
	return loader.ProcessExisting(requestProcess(query, handler), types.RequestType)
}
