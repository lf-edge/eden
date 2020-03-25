//The einfo package provides primitives for searching and processing data
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

func ParseZInfoMsg(data []byte) (ZInfoMsg info.ZInfoMsg, err error) {
	var zi info.ZInfoMsg
	err = jsonpb.UnmarshalString(string(data), &zi)
	return zi, err
}

//Print data from ZInfoMsg structure
func InfoPrn(im *info.ZInfoMsg, ds []*info.ZInfoDevSW) {
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

//Print data from ZInfoMsg structure
func ZInfoDevSWPrn(im *info.ZInfoMsg, ds []*info.ZInfoDevSW) {
	dinfo := im.GetDinfo()
	if dinfo == nil {
		return
	}
	fmt.Println("ztype:", im.GetZtype())
	fmt.Println("devId:", im.GetDevId())
	fmt.Println("dinfo.SwList:")
	for i, d := range ds {
		fmt.Printf("[%d]: %s\n", i, d)
	}
	fmt.Println("atTimeStamp:", im.GetAtTimeStamp())
	fmt.Println()
}

//Function that runs once and interrupts the workflow of InfoWatch
func HandleFirst(im *info.ZInfoMsg, ds []*info.ZInfoDevSW) bool {
	//InfoPrn(im, ds)
	ZInfoDevSWPrn(im, ds)
	return true
}

//Function that runs for all Info's selected by InfoWatch
func HandleAll(im *info.ZInfoMsg, ds []*info.ZInfoDevSW) bool {
	//InfoPrn(im, ds)
	ZInfoDevSWPrn(im, ds)
	return false
}

//HandlerFunc must process info.ZInfoMsg and return true to exit
//or false to continue
type HandlerFunc func(im *info.ZInfoMsg, ds []*info.ZInfoDevSW) bool

//QHandlerFunc must process info.ZInfoMsg with query parameters
//and return true to exit or false to continue
type QHandlerFunc func(im *info.ZInfoMsg, query map[string]string) []*info.ZInfoDevSW

//Find ZInfoMsg records with 'devid' and ZInfoDevSWF structure fields
//by reqexps in 'query'
func ZInfoDevSWFind(im *info.ZInfoMsg, query map[string]string) []*info.ZInfoDevSW {
	var dsws []*info.ZInfoDevSW

	devid, ok := query["devId"]
	if ok {
		if devid != im.DevId {
			return nil
		}
	}

	delete(query, "devId")

	dInfo := im.GetDinfo()
	if dInfo == nil {
		return nil
	}

NEXT:
	for _, d := range dInfo.SwList {
		matched := true
		var err error
		for k, v := range query {
			// Uppercase of filed's name first letter
			n := strings.Title(k)
			// Find field in structure by Titlized() name 'n'
			r := reflect.ValueOf(d)
			f := fmt.Sprint(reflect.Indirect(r).FieldByName(n))
			matched, err = regexp.Match(v, []byte(f))
			if err != nil {
				return nil
			}
			if matched == false {
				continue NEXT
			}
		}
		if matched != false {
			dsws = append(dsws, d)
		}
	}
	return dsws
}

//Function monitors the change of Info files in the 'filepath' directory with 'timeoutSeconds' according to the 'query' parameters accepted by the 'qhandler' function and subsequent processing using the 'handler' function.
func InfoWatchWithTimeout(filepath string, query map[string]string, qhandler QHandlerFunc, handler HandlerFunc, timeoutSeconds time.Duration) error {
	done := make(chan error)
	go func() {
		err := InfoWatch(filepath, query, qhandler, handler)
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

//Function monitors the change of Info files in the 'filepath' directory according to the 'query' parameters accepted by the 'qhandler' function and subsequent processing using the 'handler' function.
func InfoWatch(filepath string, query map[string]string, qhandler QHandlerFunc, handler HandlerFunc) error {
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
					ds := qhandler(&im, query)
					if ds != nil {
						if handler(&im, ds) {
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

//Function search Info files in the 'filepath' directory according to the 'query' parameters accepted by the 'qhandler' function and subsequent process using the 'handler' function.
func InfoLast(filepath string, query map[string]string, qhandler QHandlerFunc, handler HandlerFunc) error {
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
		ds := qhandler(&im, query)
		if ds != nil {
			if handler(&im, ds) {
				return nil
			}
		}
		continue
	}
	return nil
}

//Find ZInfoMsg records by reqexps in 'query' corresponded to devId and
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

func InfoChecker(dir string, q map[string]string, timeout time.Duration) (err error) {
	done := make(chan error)

	go func() {
		err = InfoWatchWithTimeout(dir, q, ZInfoDevSWFind, HandleFirst, timeout)
		done <- err
	}()
	go func() {
		handler := func(im *info.ZInfoMsg, ds []*info.ZInfoDevSW) bool {
			ZInfoDevSWPrn(im, ds)
			done <- nil
			return true
		}
		err = InfoLast(dir, q, ZInfoDevSWFind, handler)
		if err != nil {
			done <- err
		}
	}()
	return <-done
}
