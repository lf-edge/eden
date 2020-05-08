package loaders

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/lf-edge/eden/pkg/utils"
	"github.com/lf-edge/eve/api/go/info"
	"github.com/lf-edge/eve/api/go/logs"
	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
	"io"
	"net/http"
	"time"
)

const (
	StreamHeader = "X-Stream"
	StreamValue  = "true"
)

type getClient = func() *http.Client
type getUrl = func(devUUID uuid.UUID) (url string)

type remoteLoader struct {
	curCount     uint64
	lastCount    uint64
	lastTimesamp *timestamp.Timestamp
	firstLoad    bool
	devUUID      uuid.UUID
	urlLogs      getUrl
	urlInfo      getUrl
	getClient    getClient
	client       *http.Client
}

//RemoteLoader return loader from files
func RemoteLoader(getClient getClient, urlLogs getUrl, urlInfo getUrl) *remoteLoader {
	return &remoteLoader{urlLogs: urlLogs, urlInfo: urlInfo, getClient: getClient, firstLoad: true, lastTimesamp: nil, client: getClient()}
}

//Clone create copy
func (loader *remoteLoader) Clone() Loader {
	return &remoteLoader{urlLogs: loader.urlLogs, urlInfo: loader.urlInfo, getClient: loader.getClient, firstLoad: true, lastTimesamp: nil, devUUID: loader.devUUID, client: loader.getClient()}
}

func (loader *remoteLoader) getUrl(typeToProcess infoOrLogs) string {
	switch typeToProcess {
	case LogsType:
		return loader.urlLogs(loader.devUUID)
	case InfoType:
		return loader.urlInfo(loader.devUUID)
	default:
		return ""
	}
}

//SetUUID set device UUID
func (loader *remoteLoader) SetUUID(devUUID uuid.UUID) {
	loader.devUUID = devUUID
}

func (loader *remoteLoader) processNext(decoder *json.Decoder, process ProcessFunction, typeToProcess infoOrLogs) (processed, tocontinue bool, err error) {
	var buf bytes.Buffer
	switch typeToProcess {
	case LogsType:
		var emp logs.LogBundle
		if err := jsonpb.UnmarshalNext(decoder, &emp); err == io.EOF {
			return false, false, nil
		} else if err != nil {
			return false, false, err
		}
		/*if emp.Timestamp == nil {
			log.Warning("empty timestamp")
		} else if loader.lastTimesamp == nil || emp.Timestamp.Seconds > loader.lastTimesamp.Seconds {
			loader.lastTimesamp = emp.Timestamp
		} else {
			return false, true, nil
		}*/
		mler := jsonpb.Marshaler{}
		if err := mler.Marshal(&buf, &emp); err != nil {
			return false, false, err
		}
	case InfoType:
		var emp info.ZInfoMsg
		if err := jsonpb.UnmarshalNext(decoder, &emp); err == io.EOF {
			return false, false, nil
		} else if err != nil {
			return false, false, err
		}
		/*if emp.AtTimeStamp == nil {
			log.Warning("empty timestamp")
		} else if loader.lastTimesamp == nil || emp.AtTimeStamp.Seconds > loader.lastTimesamp.Seconds {
			loader.lastTimesamp = emp.AtTimeStamp
		} else {
			return false, true, nil
		}*/
		mler := jsonpb.Marshaler{}
		if err := mler.Marshal(&buf, &emp); err != nil {
			return false, false, err
		}
	}
	if loader.lastCount > loader.curCount {
		loader.curCount++
		return false, true, nil
	}
	tocontinue, err = process(buf.Bytes())
	loader.curCount++
	loader.lastCount = loader.curCount
	return true, tocontinue, err
}

func (loader *remoteLoader) process(process ProcessFunction, typeToProcess infoOrLogs, stream bool) (processed, found bool, err error) {
	u := loader.getUrl(typeToProcess)
	log.Debugf("remote controller request %s", u)
	req, err := http.NewRequest("GET", u, nil)
	if stream {
		req.Header.Add(StreamHeader, StreamValue)
	}
	response, err := loader.client.Do(req)
	if err != nil {
		return false, false, fmt.Errorf("error reading URL %s: %v", u, err)
	}
	dec := json.NewDecoder(response.Body)
	for {
		processed, doContinue, err := loader.processNext(dec, process, typeToProcess)
		if err != nil {
			return false, false, fmt.Errorf("process: %s", err)
		}
		if !doContinue {
			return processed, true, nil
		}
	}
}

func infoProcessInit(bytes []byte) (bool, error) {
	return true, nil
}

func (loader *remoteLoader) repeatableConnection(process ProcessFunction, typeToProcess infoOrLogs, stream bool) error {
	if !stream {
		loader.client.Timeout = time.Second * 10
	} else {
		loader.client.Timeout = 0
	}
	maxRepeat := utils.DefaultRepeatCount
	delayTime := utils.DefaultRepeatTimeout

repeatLoop:
	for i := 0; i < maxRepeat; i++ {
		timer := time.AfterFunc(2*delayTime, func() {
			i = 0
		})
		if stream == false {
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
				if _, _, err := loader.process(process, typeToProcess, stream); err == nil {
					return nil
				} else {
					log.Debugf("error in controller request", err)
				}
			}
		}
		timer.Stop()
		log.Infof("Attempt to re-establish connection with controller (%d) of (%d)", i, maxRepeat)
		time.Sleep(delayTime)
	}
	return fmt.Errorf("all connection attempts failed")
}

//ProcessExisting for observe existing files
func (loader *remoteLoader) ProcessExisting(process ProcessFunction, typeToProcess infoOrLogs) error {
	return loader.repeatableConnection(process, typeToProcess, false)
}

//ProcessExisting for observe new files
func (loader *remoteLoader) ProcessStream(process ProcessFunction, typeToProcess infoOrLogs, timeoutSeconds time.Duration) (err error) {
	done := make(chan error)
	if timeoutSeconds == 0 {
		timeoutSeconds = -1
	} else {
		time.AfterFunc(timeoutSeconds*time.Second, func() {
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
