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
	"reflect"
	"regexp"
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

func FindLogItem(le LogItem, query map[string]string) int {
	matched := 1
	for k, v := range query {
		// Uppercase of filed's name first letter
		n := strings.Title(k)
		// Find field in structure by Titlized() name 'n'
		r := reflect.ValueOf(le)
		f := reflect.Indirect(r).FieldByName(n).String()
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

func HandleFirst(le *LogItem) bool {
	LogPrn(le)
	return true
}

func HandleAll(le *LogItem) bool {
	LogPrn(le)
	return false
}

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

func LogWatch(filepath string, query map[string]string, handler HandlerFunc) error {
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
					time.Sleep(500 * time.Millisecond) // wait for write ends
					data, err := ioutil.ReadFile(event.Name)
					if err != nil {
						log.Fatal("Can't open", event.Name)
					}

					lb, err := ParseLogBundle(data)
					if err != nil {
						log.Print("Can't parse bundle of ", event.Name)
						log.Fatal(err)
					}

					for _, n := range lb.Log {
						//fmt.Println(n.Content)
						s := n.Content
						le, err := ParseLogItem(s)
						if err != nil {
							log.Print("Can't parse item of ", event.Name)
							log.Fatal(err)
						}
						if FindLogItem(le, query) == 1 {
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
