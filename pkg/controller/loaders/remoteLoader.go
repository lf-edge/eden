package loaders

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/lf-edge/eden/pkg/controller/cachers"
	"github.com/lf-edge/eden/pkg/controller/types"
	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eve-api/go/info"
	"github.com/lf-edge/eve-api/go/logs"
	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	//StreamHeader to pass for observe stream via http
	StreamHeader = "X-Stream"
	//StreamValue enable stream
	StreamValue = "true"
)

type getClient = func() *http.Client

// RemoteLoader implements loader from http backend of controller
type RemoteLoader struct {
	curCount     uint64
	lastCount    uint64
	lastTimesamp *timestamppb.Timestamp
	firstLoad    bool
	devUUID      uuid.UUID
	appUUID      uuid.UUID
	urlGetters   types.URLGetters
	getClient    getClient
	client       *http.Client
	cache        cachers.CacheProcessor
}

// NewRemoteLoader return loader from files
func NewRemoteLoader(getClient getClient, urlGetters types.URLGetters) *RemoteLoader {
	log.Debugf("HTTP NewRemoteLoader init")
	return &RemoteLoader{
		urlGetters:   urlGetters,
		getClient:    getClient,
		firstLoad:    true,
		lastTimesamp: nil,
		client:       getClient(),
	}
}

// SetRemoteCache add cache layer
func (loader *RemoteLoader) SetRemoteCache(cache cachers.CacheProcessor) {
	loader.cache = cache
}

// Clone create copy
func (loader *RemoteLoader) Clone() Loader {
	return &RemoteLoader{
		urlGetters:   loader.urlGetters,
		getClient:    loader.getClient,
		firstLoad:    true,
		lastTimesamp: nil,
		devUUID:      loader.devUUID,
		appUUID:      loader.appUUID,
		client:       loader.getClient(),
		cache:        loader.cache,
	}
}

func (loader *RemoteLoader) getURL(typeToProcess types.LoaderObjectType) string {
	switch typeToProcess {
	case types.LogsType:
		return loader.urlGetters.URLLogs(loader.devUUID)
	case types.InfoType:
		return loader.urlGetters.URLInfo(loader.devUUID)
	case types.MetricsType:
		return loader.urlGetters.URLMetrics(loader.devUUID)
	case types.FlowLogType:
		return loader.urlGetters.URLFlowLog(loader.devUUID)
	case types.RequestType:
		return loader.urlGetters.URLRequest(loader.devUUID)
	case types.AppsType:
		return loader.urlGetters.URLApps(loader.devUUID, loader.appUUID)
	default:
		return ""
	}
}

// SetUUID set device UUID
func (loader *RemoteLoader) SetUUID(devUUID uuid.UUID) {
	loader.devUUID = devUUID
}

// SetAppUUID set app UUID
func (loader *RemoteLoader) SetAppUUID(appUUID uuid.UUID) {
	loader.appUUID = appUUID
}

func (loader *RemoteLoader) processNext(decoder *json.Decoder, process ProcessFunction, typeToProcess types.LoaderObjectType, stream bool) (processed, tocontinue bool, err error) {
	var buf []byte
	switch typeToProcess {
	case types.LogsType:
		var emp logs.LogBundle
		if err := decoder.Decode(&emp); err == io.EOF {
			return false, false, nil
		} else if err != nil {
			return false, false, err
		}
		if buf, err = protojson.Marshal(&emp); err != nil {
			return false, false, err
		}
	case types.InfoType:
		var emp info.ZInfoMsg
		if err := decoder.Decode(&emp); err == io.EOF {
			return false, false, nil
		} else if err != nil {
			return false, false, err
		}
		if buf, err = protojson.Marshal(&emp); err != nil {
			return false, false, err
		}
	}
	if loader.cache != nil {
		if err = loader.cache.CheckAndSave(loader.devUUID, typeToProcess, buf); err != nil {
			log.Errorf("error in cache: %s", err)
		}
	}
	if loader.lastCount > loader.curCount {
		loader.curCount++
		return false, true, nil
	}
	tocontinue, err = process(buf)
	if stream {
		time.Sleep(1 * time.Second) //wait for load all data from buffer
	}
	loader.curCount++
	loader.lastCount = loader.curCount
	return true, tocontinue, err
}

func (loader *RemoteLoader) process(process ProcessFunction, typeToProcess types.LoaderObjectType, stream bool) (processed, found bool, err error) {
	u := loader.getURL(typeToProcess)
	log.Debugf("remote controller request %s", u)
	req, _ := http.NewRequest("GET", u, nil)
	if stream {
		req.Header.Add(StreamHeader, StreamValue)
	}
	response, err := loader.client.Do(req)
	if err != nil {
		return false, false, fmt.Errorf("error reading URL %s: %v", u, err)
	}
	dec := json.NewDecoder(response.Body)
	for {
		processed, doContinue, err := loader.processNext(dec, process, typeToProcess, stream)
		if err != nil {
			return false, false, fmt.Errorf("process: %s", err)
		}
		if !doContinue {
			return processed, true, nil
		}
	}
}

func infoProcessInit(_ []byte) (bool, error) {
	return true, nil
}

func (loader *RemoteLoader) repeatableConnection(process ProcessFunction, typeToProcess types.LoaderObjectType, stream bool) error {
	if !stream {
		loader.client.Timeout = time.Second * 10
	} else {
		loader.client.Timeout = 0
	}
	maxRepeat := defaults.DefaultRepeatCount
	delayTime := defaults.DefaultRepeatTimeout

repeatLoop:
	for i := 0; i < maxRepeat; i++ {
		timer := time.AfterFunc(2*delayTime, func() {
			i = 0
		})
		if !stream {
			if _, _, err := loader.process(process, typeToProcess, false); err == nil {
				return nil
			}
		} else {
			if loader.firstLoad {
				if _, _, err := loader.process(infoProcessInit, typeToProcess, false); err == nil {
					loader.firstLoad = false
					goto repeatLoop
				}
			} else {
				if i > 0 { //load existing elements for repeat
					loader.curCount = 0
					if processed, _, err := loader.process(process, typeToProcess, false); err == nil {
						if processed {
							return nil
						}
					}
				}
				if _, _, err := loader.process(process, typeToProcess, stream); err != nil {
					log.Debugf("error in controller request: %s", err)
				} else {
					return nil
				}
			}
		}
		timer.Stop()
		log.Infof("Attempt to re-establish connection with controller (%d) of (%d)", i, maxRepeat)
		time.Sleep(delayTime)
	}
	return fmt.Errorf("all connection attempts failed")
}

// ProcessExisting for observe existing files
func (loader *RemoteLoader) ProcessExisting(process ProcessFunction, typeToProcess types.LoaderObjectType) error {
	return loader.repeatableConnection(process, typeToProcess, false)
}

// ProcessStream for observe new files
func (loader *RemoteLoader) ProcessStream(process ProcessFunction, typeToProcess types.LoaderObjectType, timeoutSeconds time.Duration) (err error) {
	done := make(chan error)
	if timeoutSeconds != 0 {
		time.AfterFunc(timeoutSeconds, func() {
			done <- fmt.Errorf("timeout")
		})
	}

	go func() {
		done <- loader.repeatableConnection(process, typeToProcess, true)
	}()
	err = <-done
	loader.client.CloseIdleConnections()
	return err
}
