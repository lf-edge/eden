package loaders

import (
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/lf-edge/eden/pkg/controller/cachers"
	"github.com/lf-edge/eden/pkg/controller/types"
	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"path"
	"sort"
	"time"
)

type fileLoader struct {
	appUUID uuid.UUID
	devUUID uuid.UUID
	getters types.DirGetters
	cache   cachers.CacheProcessor
}

//FileLoader return loader from files
func FileLoader(getters types.DirGetters) *fileLoader {
	log.Debugf("FileLoader init")
	return &fileLoader{getters: getters}
}

//SetRemoteCache add cache layer
func (loader *fileLoader) SetRemoteCache(cache cachers.CacheProcessor) {
	loader.cache = cache
}

//Clone create copy
func (loader *fileLoader) Clone() Loader {
	return &fileLoader{
		getters: loader.getters,
		devUUID: loader.devUUID,
		appUUID: loader.appUUID,
		cache:   loader.cache,
	}
}

func (loader *fileLoader) getFilePath(typeToProcess types.LoaderObjectType) string {
	switch typeToProcess {
	case types.LogsType:
		return loader.getters.LogsGetter(loader.devUUID)
	case types.InfoType:
		return loader.getters.InfoGetter(loader.devUUID)
	case types.MetricsType:
		return loader.getters.MetricsGetter(loader.devUUID)
	case types.RequestType:
		return loader.getters.RequestGetter(loader.devUUID)
	case types.AppsType:
		return loader.getters.AppsGetter(loader.devUUID, loader.appUUID)
	default:
		return ""
	}
}

//SetUUID set device UUID
func (loader *fileLoader) SetUUID(devUUID uuid.UUID) {
	loader.devUUID = devUUID
}

//SetUUID set app UUID
func (loader *fileLoader) SetAppUUID(appUUID uuid.UUID) {
	loader.appUUID = appUUID
}

//ProcessExisting for observe existing files
func (loader *fileLoader) ProcessExisting(process ProcessFunction, typeToProcess types.LoaderObjectType) error {
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
		if loader.cache != nil {
			if err = loader.cache.CheckAndSave(loader.devUUID, typeToProcess, data); err != nil {
				log.Errorf("error in cache: %s", err)
			}
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
func (loader *fileLoader) ProcessStream(process ProcessFunction, typeToProcess types.LoaderObjectType, timeoutSeconds time.Duration) error {
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
					if loader.cache != nil {
						if err = loader.cache.CheckAndSave(loader.devUUID, typeToProcess, data); err != nil {
							log.Errorf("error in cache: %s", err)
						}
					}
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
