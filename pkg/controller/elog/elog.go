//Package elog provides primitives for searching and processing data
//in Log files.
package elog

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/golang/protobuf/jsonpb"
	"github.com/lf-edge/eve/api/go/logs"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"path"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"time"
)

//LogItem is the structure for saving log fields
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

//ParseLogBundle unmarshal LogBundle
func ParseLogBundle(data []byte) (logBundle logs.LogBundle, err error) {
	var lb logs.LogBundle
	err = jsonpb.UnmarshalString(string(data), &lb)
	return lb, err
}

//ParseLogItem apply regexp on logItem
func ParseLogItem(data string) (logItem LogItem, err error) {
	re := regexp.MustCompile(`(?P<time>[^{]*): (?P<json>{.*})`)
	parts := re.SubexpNames()
	result := re.FindAllStringSubmatch(data, -1)
	m := map[string]string{}
	for i, n := range result[0] {
		m[parts[i]] = n
	}
	var le LogItem
	err = json.Unmarshal([]byte(m["json"]), &le)

	return le, err
}

//LogItemFind find LogItem records by reqexps in 'query' corresponded to LogItem structure.
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

//HandleFirst runs once and interrupts the workflow of LogWatch
func HandleFirst(le *LogItem) bool {
	LogPrn(le)
	return true
}

//HandleAll runs for all Logs selected by LogWatch
func HandleAll(le *LogItem) bool {
	LogPrn(le)
	return false
}

//LogPrn print Log data
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

//LogWatch monitors the change of Log files in the 'filepath' directory
//according to the 'query' reqexps and processing using the 'handler' function.
func LogWatch(filepath string, query map[string]string, handler HandlerFunc, timeoutSeconds time.Duration) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	devID, ok := query["devId"]
	if ok {
		delete(query, "devId")
	}
	eveVersion, ok := query["eveVersion"]
	if ok {
		delete(query, "eveVersion")
	}
	if timeoutSeconds == 0 {
		timeoutSeconds = -1
	}
	done := make(chan error)
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					done <- errors.New("watcher closed")
					return
				}
				switch event.Op {
				case fsnotify.Write:
					time.Sleep(1 * time.Second) // wait for write ends
					data, err := ioutil.ReadFile(event.Name)
					if err != nil {
						log.Error("Can't open ", event.Name)
						continue
					}
					log.Debugf("parse log file %s", event.Name)

					lb, err := ParseLogBundle(data)
					if err != nil {
						log.Error("Can't parse bundle of ", event.Name)
						continue
					}
					if devID != "" && devID != lb.DevID {
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
							log.Error("Can't parse item in ", event.Name)
							continue
						}
						if LogItemFind(le, query) == 1 {
							if handler(&le) {
								done <- nil
								return
							}
						}
					}
					continue
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					done <- err
					return
				}
				log.Errorf("error: %s", err)
			case <-time.After(timeoutSeconds * time.Second):
				done <- errors.New("timeout")
				return
			}
		}
	}()

	err = watcher.Add(filepath)
	if err != nil {
		return err
	}

	err = <-done
	_ = watcher.Close()
	return err
}

//LogLast function process Log files in the 'filepath' directory
//according to the 'query' reqexps and return last founded item
func LogLast(filepath string, query map[string]string, handler HandlerFunc) error {
	devID, ok := query["devId"]
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
		log.Debugf("parse log file %s", fileFullPath)
		data, err := ioutil.ReadFile(fileFullPath)
		if err != nil {
			log.Error("Can't open ", fileFullPath)
			continue
		}

		lb, err := ParseLogBundle(data)
		if err != nil {
			log.Error("Can't parse bundle of ", fileFullPath)
			continue
		}
		if devID != "" && devID != lb.DevID {
			continue
		}
		if eveVersion != "" && eveVersion != lb.EveVersion {
			continue
		}
		for _, n := range lb.Log {
			s := n.Content
			le, err := ParseLogItem(s)
			if err != nil {
				log.Error("Can't parse items in ", file.Name())
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

//LogChecker check logs by pattern from existence files with LogLast and use LogWatchWithTimeout with timeout for observe new files
func LogChecker(dir string, q map[string]string, timeout time.Duration) (err error) {
	log.Infof("wait for log in %s", dir)
	done := make(chan error)
	go func() {
		done <- LogWatch(dir, q, HandleFirst, timeout)
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
