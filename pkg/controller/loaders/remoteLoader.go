package loaders

import (
	"bytes"
	"encoding/json"
	"github.com/gogo/protobuf/proto"
	"github.com/golang/protobuf/jsonpb"
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

type getUrl = func(devUUID uuid.UUID) (url string)

type remoteLoader struct {
	devUUID uuid.UUID
	urlLogs getUrl
	urlInfo getUrl
	client  *http.Client
}

//RemoteLoader return loader from files
func RemoteLoader(client *http.Client, urlLogs getUrl, urlInfo getUrl) *remoteLoader {
	return &remoteLoader{urlLogs: urlLogs, urlInfo: urlInfo, client: client}
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

func processNext(decoder *json.Decoder, emp proto.Message, process ProcessFunction) (bool, error) {
	if err := jsonpb.UnmarshalNext(decoder, emp); err == io.EOF {
		return false, nil
	} else if err != nil {
		return false, err
	}
	var buf bytes.Buffer
	mler := jsonpb.Marshaler{}
	if err := mler.Marshal(&buf, emp); err != nil {
		return false, err
	}
	return process(buf.Bytes())
}

func (loader *remoteLoader) process(process ProcessFunction, typeToProcess infoOrLogs, stream bool) error {
	u := loader.getUrl(typeToProcess)
	log.Debugf("remote controller request %s", u)
	req, err := http.NewRequest("GET", u, nil)
	if stream {
		req.Header.Add(StreamHeader, StreamValue)
	}
	response, err := loader.client.Do(req)
	if err != nil {
		log.Fatalf("error reading URL %s: %v", u, err)
	}
	dec := json.NewDecoder(response.Body)
	for {
		doContinue := true
		switch typeToProcess {
		case LogsType:
			var emp logs.LogBundle
			doContinue, err = processNext(dec, &emp, process)
		case InfoType:
			var emp info.ZInfoMsg
			doContinue, err = processNext(dec, &emp, process)
		}
		if err != nil {
			log.Fatalf("process: %s", err)
		}
		if !doContinue {
			return nil
		}
	}
}

//ProcessExisting for observe existing files
func (loader *remoteLoader) ProcessExisting(process ProcessFunction, typeToProcess infoOrLogs) error {
	loader.client.Timeout = time.Second * 10
	return loader.process(process, typeToProcess, false)
}

//ProcessExisting for observe new files
func (loader *remoteLoader) ProcessStream(process ProcessFunction, typeToProcess infoOrLogs, timeoutSeconds time.Duration) error {
	if timeoutSeconds > 0 {
		loader.client.Timeout = time.Second * timeoutSeconds
	} else {
		loader.client.Timeout = time.Second * 10
	}
	return loader.process(process, typeToProcess, true)
}
