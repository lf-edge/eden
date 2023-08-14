// Package erequest provides primitives for searching and processing data
// in Request files.
package erequest

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"regexp"
	"strings"

	"github.com/lf-edge/eden/pkg/controller/loaders"
	"github.com/lf-edge/eden/pkg/controller/types"
	"github.com/lf-edge/eden/pkg/utils"
	log "github.com/sirupsen/logrus"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// RequestFormat the format to print output Requests
type RequestFormat byte

const (
	//RequestLines returns requests line by line
	RequestLines RequestFormat = iota
	//RequestJSON returns requests in JSON format
	RequestJSON
)

// ParseRequestItem apply regexp on APIRequest
func ParseRequestItem(data []byte) (logItem *types.APIRequest, err error) {
	var le types.APIRequest
	err = json.Unmarshal(data, &le)
	return &le, err
}

// RequestItemFind find APIRequest records by reqexps in 'query' corresponded to APIRequest structure.
func RequestItemFind(le *types.APIRequest, query map[string]string) bool {
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

// RequestPrn print APIRequest data
func RequestPrn(le *types.APIRequest, format RequestFormat) {
	switch format {
	case RequestJSON:
		enc := json.NewEncoder(os.Stdout)
		_ = enc.Encode(le)
	case RequestLines:
		fmt.Println("uuid:", le.UUID)
		fmt.Println("client-ip:", le.ClientIP)
		fmt.Println("forwarded:", le.Forwarded)
		fmt.Println("method:", le.Method)
		fmt.Println("url:", le.URL)
		fmt.Println("timestamp:", le.Timestamp)
		fmt.Println()
	default:
		log.Errorf("unknown log format requested")
	}
}

// HandlerFunc must process APIRequest and return true to exit
// or false to continue
type HandlerFunc func(request *types.APIRequest) bool

func requestProcess(query map[string]string, handler HandlerFunc) loaders.ProcessFunction {
	return func(bytes []byte) (bool, error) {
		le, err := ParseRequestItem(bytes)
		if err != nil {
			log.Debugf("logProcess: %s", err)
		}
		if RequestItemFind(le, query) {
			if handler(le) {
				return false, nil
			}
		}
		return true, nil
	}
}

// RequestLast function process Request files in the 'filepath' directory
// according to the 'query' reqexps and return last founded item
func RequestLast(loader loaders.Loader, query map[string]string, handler HandlerFunc) error {
	return loader.ProcessExisting(requestProcess(query, handler), types.RequestType)
}
