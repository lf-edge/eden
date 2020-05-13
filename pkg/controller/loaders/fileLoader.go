package loaders

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

type getDir = func(devUUID uuid.UUID) (dir string)

type fileLoader struct {
	devUUID    uuid.UUID
	logsGetter getDir
	infoGetter getDir
}

//FileLoader return loader from files
func FileLoader(logsGetter getDir, infoGetter getDir) *fileLoader {
	log.Debugf("FileLoader init")
	return &fileLoader{logsGetter: logsGetter, infoGetter: infoGetter}
}

//Clone create copy
func (loader *fileLoader) Clone() Loader {
	return &fileLoader{logsGetter: loader.logsGetter, infoGetter: loader.infoGetter, devUUID: loader.devUUID}
}

func (loader *fileLoader) getFilePath(typeToProcess infoOrLogs) string {
	switch typeToProcess {
	case LogsType:
		return loader.logsGetter(loader.devUUID)
	case InfoType:
		return loader.infoGetter(loader.devUUID)
	default:
		return ""
	}
}

//SetUUID set device UUID
func (loader *fileLoader) SetUUID(devUUID uuid.UUID) {
	loader.devUUID = devUUID
}

//ProcessExisting for observe existing files
func (loader *fileLoader) ProcessExisting(process ProcessFunction, typeToProcess infoOrLogs) error {
	files, err := ioutil.ReadDir(loader.getFilePath(typeToProcess))
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
		fileFullPath := path.Join(loader.getFilePath(typeToProcess), file.Name())
		log.Debugf("local controller parse %s", fileFullPath)
		data, err := ioutil.ReadFile(fileFullPath)
		if err != nil {
			log.Error("Can't open ", fileFullPath)
			continue
		}
		doContinue, err := process(data)
		if err != nil {
			return err
		}
		if !doContinue {
			return nil
		}
	}
	return nil
}

//ProcessExisting for observe new files
func (loader *fileLoader) ProcessStream(process ProcessFunction, typeToProcess infoOrLogs, timeoutSeconds time.Duration) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	done := make(chan error)

	if timeoutSeconds == 0 {
		timeoutSeconds = -1
	} else {
		time.AfterFunc(timeoutSeconds*time.Second, func() {
			done <- fmt.Errorf("timeout")
		})
	}
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
					log.Debugf("local controller parse %s", event.Name)
					doContinue, err := process(data)
					if err != nil {
						done <- err
					}
					if !doContinue {
						done <- nil
						return
					}
					continue
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					done <- err
					return
				}
				log.Errorf("error: %s", err)
			}
		}
	}()

	err = watcher.Add(loader.getFilePath(typeToProcess))
	if err != nil {
		return err
	}

	err = <-done
	_ = watcher.Close()
	return err
}
