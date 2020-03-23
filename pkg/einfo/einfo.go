package einfo

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"time"
	"github.com/fsnotify/fsnotify"
	"github.com/golang/protobuf/jsonpb"
	"github.com/lf-edge/eve/api/go/info"
)

func ParseZInfoMsg(data []byte) (ZInfoMsg info.ZInfoMsg, err error) {
	var zi  info.ZInfoMsg
	err = jsonpb.UnmarshalString(string(data), &zi)
	return zi, err
}
	

func InfoPrn(im *info.ZInfoMsg) {
	fmt.Println("ztype:", im.GetZtype())
	fmt.Println("devId:", im.GetDevId())
	if (im.GetDinfo() != nil) {
		fmt.Println("dinfo:", im.GetDinfo())
	}
	if (im.GetAinfo() != nil) {
		fmt.Println("ainfo:", im.GetAinfo())
	}
	if (im.GetNiinfo() != nil) {
		fmt.Println("niinfo:", im.GetNiinfo())
	}
	fmt.Println("atTimeStamp:", im.GetAtTimeStamp())
	fmt.Println()
}


func HandleFirst(im *info.ZInfoMsg) bool {
	InfoPrn(im)
	return true
}

func HandleAll(im *info.ZInfoMsg) bool {
	InfoPrn(im)
	return false
}

//HandlerFunc must process info.ZInfoMsg and return true to exit
//or false to continue
type HandlerFunc func(*info.ZInfoMsg) bool

func InfoWatchWithTimeout(filepath string, handler HandlerFunc, timeoutSeconds time.Duration) error {
	done := make(chan error)
	go func() {
		err := InfoWatch(filepath, handler)
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

func InfoWatch(filepath string, handler HandlerFunc) error {
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

					im, err := ParseZInfoMsg(data)
					if err != nil {
						log.Print("Can't parse ZInfoMsg", event.Name)
						log.Fatal(err)
					}
					if handler(&im) {
						return
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
