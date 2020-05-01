package einfo

import (
	"fmt"
	"github.com/fsnotify/fsnotify"
	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"path"
	"sort"
	"time"
)

type getInfoDir = func(devUUID uuid.UUID) (dir string)

type fileLoader struct {
	devUUID        uuid.UUID
	filePathGetter getInfoDir
}

//FileLoader return loader from files
func FileLoader(filePathGetter getInfoDir) *fileLoader {
	return &fileLoader{filePathGetter: filePathGetter}
}

func (loader *fileLoader) getFilePath() string {
	return loader.filePathGetter(loader.devUUID)
}

func (loader *fileLoader) SetUUID(devUUID uuid.UUID) {
	loader.devUUID = devUUID
}

//InfoWatch monitors the change of Info files in the 'filepath' directory according to the 'query' parameters accepted by the 'qhandler' function and subsequent processing using the 'handler' function with 'timeoutSeconds'.
func (loader *fileLoader) InfoWatch(query map[string]string, qhandler QHandlerFunc, handler HandlerFunc, infoType ZInfoType, timeoutSeconds time.Duration) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
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
					done <- fmt.Errorf("watcher closed")
					return
				}
				switch event.Op {
				case fsnotify.Write:
					time.Sleep(1 * time.Second) // wait for write ends
					data, err := ioutil.ReadFile(event.Name)
					if err != nil {
						log.Error("Can't open", event.Name)
						continue
					}
					log.Debugf("parse info file %s", event.Name)

					im, err := ParseZInfoMsg(data)
					if err != nil {
						log.Error("Can't parse ZInfoMsg", event.Name)
						continue
					}
					ds := qhandler(&im, query, infoType)
					if ds != nil {
						if handler(&im, ds, infoType) {
							done <- nil
							return
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
				done <- fmt.Errorf("timeout")
				return
			}
		}
	}()

	err = watcher.Add(loader.getFilePath())
	if err != nil {
		return err
	}

	err = <-done
	_ = watcher.Close()
	return err
}

//InfoLast search Info files in the 'filepath' directory according to the 'query' parameters accepted by the 'qhandler' function and subsequent process using the 'handler' function.
func (loader *fileLoader) InfoLast(query map[string]string, qhandler QHandlerFunc, handler HandlerFunc, infoType ZInfoType) error {
	files, err := ioutil.ReadDir(loader.getFilePath())
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
		fileFullPath := path.Join(loader.getFilePath(), file.Name())
		log.Debugf("parse info file %s", fileFullPath)
		data, err := ioutil.ReadFile(fileFullPath)
		if err != nil {
			log.Error("Can't open ", fileFullPath)
			continue
		}

		im, err := ParseZInfoMsg(data)
		if err != nil {
			log.Error("Can't parse ZInfoMsg ", fileFullPath)
			continue
		}
		ds := qhandler(&im, query, infoType)
		if ds != nil {
			if handler(&im, ds, infoType) {
				return nil
			}
		}
		continue
	}
	return nil
}
