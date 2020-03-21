package elog

import (
	"encoding/json"
	"fmt"
	"log"
	timestamp "github.com/golang/protobuf/ptypes/timestamp"
	"regexp"
	"reflect"
	"strings"
	"io/ioutil"
	"github.com/fsnotify/fsnotify"
)

// from eve/api/go/logs/log.pb.go
type LogBundle struct {
        DevID                string               `protobuf:"bytes,1,opt,name=devID,proto3" json:"devID,omitempty"`
        Image                string               `protobuf:"bytes,2,opt,name=image,proto3" json:"image,omitempty"`
        Log                  []*LogEntry          `protobuf:"bytes,3,rep,name=log,proto3" json:"log,omitempty"`
        Timestamp            *timestamp.Timestamp `protobuf:"bytes,4,opt,name=timestamp,proto3" json:"timestamp,omitempty"`
        EveVersion           string               `protobuf:"bytes,5,opt,name=eveVersion,proto3" json:"eveVersion,omitempty"`
        XXX_NoUnkeyedLiteral struct{}             `json:"-"`
        XXX_unrecognized     []byte               `json:"-"`
        XXX_sizecache        int32                `json:"-"`
}

// from eve/api/go/logs/log.pb.go
type LogEntry struct {
        Severity             string               `protobuf:"bytes,1,opt,name=severity,proto3" json:"severity,omitempty"`
        Source               string               `protobuf:"bytes,2,opt,name=source,proto3" json:"source,omitempty"`
        Iid                  string               `protobuf:"bytes,3,opt,name=iid,proto3" json:"iid,omitempty"`
        Content              string               `protobuf:"bytes,4,opt,name=content,proto3" json:"content,omitempty"`
        Msgid                uint64               `protobuf:"varint,5,opt,name=msgid,proto3" json:"msgid,omitempty"`
        Tags                 map[string]string    `protobuf:"bytes,6,rep,name=tags,proto3" json:"tags,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
        Timestamp            *timestamp.Timestamp `protobuf:"bytes,7,opt,name=timestamp,proto3" json:"timestamp,omitempty"`
        Filename             string               `protobuf:"bytes,8,opt,name=filename,proto3" json:"filename,omitempty"`
        Function             string               `protobuf:"bytes,9,opt,name=function,proto3" json:"function,omitempty"`
        XXX_NoUnkeyedLiteral struct{}             `json:"-"`
        XXX_unrecognized     []byte               `json:"-"`
        XXX_sizecache        int32                `json:"-"`
}

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


func Parse_LogBundle(data []byte) LogBundle {
	var lb LogBundle
	json.Unmarshal([]byte(string(data)), &lb)
	return lb
}

func Parse_LogItem(data string) LogItem {
	re := regexp.MustCompile(`(?P<time>[^{]*): (?P<json>\{.*\})`)
	parts := re.SubexpNames()
	result := re.FindAllStringSubmatch(string(data), -1)
        m := map[string]string{}
        for i, n := range result[0] {
            m[parts[i]] = n
        }
	//fmt.Println("time: ", m["time"])
	//fmt.Println("json: ", m["json"])

	var le LogItem
	json.Unmarshal([]byte(string(m["json"])), &le)

	return le
}

func Find_LogItem(le LogItem, query map[string] string) int {
	matched := 1
        for k, v := range query {
		// Uppercase of filed's name first letter
		n := strings.Title(k)
		// Find field in structure by Titlized() name 'n'
		r := reflect.ValueOf(le)
		f := reflect.Indirect(r).FieldByName(n).String()
		matched, err := regexp.Match(v, []byte(f))
		if (err != nil) {
			return(-1)
		}
		if (matched == false) {
			return(0)
		}
	}
	return(matched)
}

func Log_prn(le *LogItem) {
	fmt.Println("source:",le.Source)
	fmt.Println("level:",le.Level)
	fmt.Println("msg:",le.Msg)
	fmt.Println("file:",le.File)
	fmt.Println("func:",le.Func)
	fmt.Println("time:",le.Time)
	fmt.Println("pid:",le.Pid)
	fmt.Println("partition:",le.Partition)
	fmt.Println()
}

type Handler_func func(*LogItem)

func Log_watch(filepath string, query map[string] string, handler Handler_func) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	done := make(chan bool)
	go func() {
	MONITOR:
		for {
			select {
			case event := <-watcher.Events:
				switch event.Op {
				case fsnotify.Write:
					data, err := ioutil.ReadFile(event.Name)
					if err != nil {
						log.Fatal("Can't open", event.Name)
						break MONITOR
					}

					lb := Parse_LogBundle(data)

					for _, n := range lb.Log {
						//fmt.Println(n.Content)
						s := string(n.Content)
						le := Parse_LogItem(s)
						if (Find_LogItem(le, query) == 1) {
							handler(&le)
						}
					}
					continue
				}
			case err := <-watcher.Errors:
				fmt.Println("Error:", err)
			}
		}
	}()

	err = watcher.Add(filepath)
	if err != nil {
		log.Fatal(err)
	}

	<-done
}

