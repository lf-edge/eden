package loaders

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"sort"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/lf-edge/eden/pkg/controller/cachers"
	"github.com/lf-edge/eden/pkg/controller/types"
	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
)

// FileLoader implements loader from file backend of controller
type FileLoader struct {
	appUUID uuid.UUID
	devUUID uuid.UUID
	getters types.DirGetters
	cache   cachers.CacheProcessor
}

// NewFileLoader return loader from files
func NewFileLoader(getters types.DirGetters) *FileLoader {
	log.Debugf("NewFileLoader init")
	return &FileLoader{getters: getters}
}

// SetRemoteCache add cache layer
func (loader *FileLoader) SetRemoteCache(cache cachers.CacheProcessor) {
	loader.cache = cache
}

// Clone create copy
func (loader *FileLoader) Clone() Loader {
	return &FileLoader{
		getters: loader.getters,
		devUUID: loader.devUUID,
		appUUID: loader.appUUID,
		cache:   loader.cache,
	}
}

func (loader *FileLoader) getFilePath(typeToProcess types.LoaderObjectType) string {
	switch typeToProcess {
	case types.LogsType:
		return loader.getters.LogsGetter(loader.devUUID)
	case types.InfoType:
		return loader.getters.InfoGetter(loader.devUUID)
	case types.MetricsType:
		return loader.getters.MetricsGetter(loader.devUUID)
	case types.FlowLogType:
		return loader.getters.FlowLogGetter(loader.devUUID)
	case types.RequestType:
		return loader.getters.RequestGetter(loader.devUUID)
	case types.AppsType:
		return loader.getters.AppsGetter(loader.devUUID, loader.appUUID)
	default:
		return ""
	}
}

// SetUUID set device UUID
func (loader *FileLoader) SetUUID(devUUID uuid.UUID) {
	loader.devUUID = devUUID
}

// SetAppUUID set app UUID
func (loader *FileLoader) SetAppUUID(appUUID uuid.UUID) {
	loader.appUUID = appUUID
}

// ProcessExisting for observe existing files
func (loader *FileLoader) ProcessExisting(process ProcessFunction, typeToProcess types.LoaderObjectType) error {
	entries, err := os.ReadDir(loader.getFilePath(typeToProcess))
	if err != nil {
		return err
	}
	files := make([]fs.FileInfo, 0, len(entries))
	for _, eachFile := range entries {
		fInfo, err := eachFile.Info()
		if err != nil {
			return err
		}
		files = append(files, fInfo)
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
		data, err := os.ReadFile(fileFullPath)
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

// ProcessStream for observe new files
func (loader *FileLoader) ProcessStream(process ProcessFunction, typeToProcess types.LoaderObjectType, timeoutSeconds time.Duration) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	done := make(chan error)

	if timeoutSeconds != 0 {
		time.AfterFunc(timeoutSeconds, func() {
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
					data, err := os.ReadFile(event.Name)
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
