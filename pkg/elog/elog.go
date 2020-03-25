//The einfo package provides primitives for searching and processing data
//in Log files.
package elog

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/golang/protobuf/jsonpb"
	"github.com/lf-edge/eve/api/go/logs"
	"io/ioutil"
	"log"
	"path"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"time"
)

type LogItem struct {
	Source    string
	Level     string
	Msg       string
	File      string
	Func      string
	Time      string
	Pid       string
	Partition string
}

func ParseLogBundle(data []byte) (logBundle logs.LogBundle, err error) {
	var lb logs.LogBundle
	err = jsonpb.UnmarshalString(string(data), &lb)
	return lb, err
}

func ParseLogItem(data string) (logItem LogItem, err error) {
	re := regexp.MustCompile(`(?P<time>[^{]*): (?P<json>{.*})`)
	parts := re.SubexpNames()
	result := re.FindAllStringSubmatch(data, -1)
	m := map[string]string{}
	for i, n := range result[0] {
		m[parts[i]] = n
	}
	//fmt.Println("time: ", m["time"])
	//fmt.Println("json: ", m["json"])

	var le LogItem
	err = json.Unmarshal([]byte(m["json"]), &le)

	return le, err
}

//Find LogItem records by reqexps in 'query' corresponded to LogItem structure.
func LogItemFind(le LogItem, query map[string]string) int {
	matched := 1
	for k, v := range query {
		// Uppercase of filed's name first letter
		n := strings.Title(k)
		// Find field in structure by Titlized() name 'n'
		r := reflect.ValueOf(le)
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

//Function that runs once and interrupts the workflow of LogWatch
func HandleFirst(le *LogItem) bool {
	LogPrn(le)
	return true
}

//Function that runs for all Logs selected by LogWatch
func HandleAll(le *LogItem) bool {
	LogPrn(le)
	return false
}

//Print Log data
func LogPrn(le *LogItem) {
	fmt.Println("source:", le.Source)
	fmt.Println("level:", le.Level)
	fmt.Println("msg:", le.Msg)
	fmt.Println("file:", le.File)
	fmt.Println("func:", le.Func)
	fmt.Println("time:", le.Time)
	fmt.Println("pid:", le.Pid)
	fmt.Println("partition:", le.Partition)
	fmt.Println()
}

//HandlerFunc must process LogItem and return true to exit
//or false to continue
type HandlerFunc func(*LogItem) bool

//Function monitors the change of Log files in the 'filepath' directory
//with 'timeoutSeconds' according to the 'query' reqexps and
//processing using the 'handler' function.
func LogWatchWithTimeout(filepath string, query map[string]string, handler HandlerFunc, timeoutSeconds time.Duration) error {
	done := make(chan error)
	go func() {
		err := LogWatch(filepath, query, handler)
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

//Function monitors the change of Log files in the 'filepath' directory
//according to the 'query' reqexps and processing using the 'handler' function.
func LogWatch(filepath string, query map[string]string, handler HandlerFunc) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()
	devId, ok := query["devId"]
	if ok {
		delete(query, "devId")
	}
	eveVersion, ok := query["eveVersion"]
	if ok {
		delete(query, "eveVersion")
	}
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
						log.Print("Can't open ", event.Name)
						continue
					}

					lb, err := ParseLogBundle(data)
					if err != nil {
						log.Print("Can't parse bundle of ", event.Name)
						continue
					}
					if devId != "" && devId != lb.DevID {
						continue
					}
					if eveVersion != "" && eveVersion != lb.EveVersion {
						continue
					}
					for _, n := range lb.Log {
						//fmt.Println(n.Content)
						s := n.Content
						le, err := ParseLogItem(s)
						if err != nil {
							log.Print("Can't parse item in ", event.Name)
							continue
						}
						if LogItemFind(le, query) == 1 {
							if handler(&le) {
								return
							}
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

//Function process Log files in the 'filepath' directory
//according to the 'query' reqexps and return last founded item
func LogLast(filepath string, query map[string]string, handler HandlerFunc) error {
	devId, ok := query["devId"]
	if ok {
		delete(query, "devId")
	}
	eveVersion, ok := query["eveVersion"]
	if ok {
		delete(query, "eveVersion")
	}
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

		lb, err := ParseLogBundle(data)
		if err != nil {
			log.Print("Can't parse bundle of ", fileFullPath)
			continue
		}
		if devId != "" && devId != lb.DevID {
			continue
		}
		if eveVersion != "" && eveVersion != lb.EveVersion {
			continue
		}
		for _, n := range lb.Log {
			s := n.Content
			le, err := ParseLogItem(s)
			if err != nil {
				log.Print("Can't parse items in ", file.Name())
				continue
			}
			if LogItemFind(le, query) == 1 {
				if handler(&le) {
					return nil
				}
			}
		}
		continue
	}
	return nil
}

func LogChecker(dir string, q map[string]string, timeout time.Duration) (err error) {
	done := make(chan error)
	go func() {
		done <- LogWatchWithTimeout(dir, q, HandleFirst, timeout)
	}()
	go func() {
		handler := func(item *LogItem) bool {
			done <- nil
			return HandleFirst(item)
		}
		err := LogLast(dir, q, handler)
		if err != nil {
			done <- err
		}
	}()
	return <-done
}
