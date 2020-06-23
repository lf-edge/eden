package cachers

import (
	"bytes"
	"fmt"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/lf-edge/eve/api/go/info"
	"github.com/lf-edge/eve/api/go/logs"
	"github.com/lf-edge/eve/api/go/metrics"
	uuid "github.com/satori/go.uuid"
	"io/ioutil"
	"os"
	"path/filepath"
)

type getDir = func(devUUID uuid.UUID) (dir string)

type fileCache struct {
	dirLogs    getDir
	dirInfo    getDir
	dirMetrics getDir
}

func FileCache(dirLogs getDir, dirInfo getDir, dirMetrics getDir) *fileCache {
	return &fileCache{
		dirLogs:    dirLogs,
		dirInfo:    dirInfo,
		dirMetrics: dirMetrics,
	}
}

func (cacher *fileCache) CheckAndSave(devUUID uuid.UUID, typeToProcess int, data []byte) error {
	var pathToCheck string
	var itemTimeStamp *timestamp.Timestamp
	var buf bytes.Buffer
	buf.Write(data)
	switch typeToProcess {
	case int(LogsType):
		pathToCheck = cacher.dirLogs(devUUID)
		var emp logs.LogBundle
		if err := jsonpb.Unmarshal(&buf, &emp); err != nil {
			return err
		}
		itemTimeStamp = emp.Timestamp
	case int(InfoType):
		pathToCheck = cacher.dirInfo(devUUID)
		var emp info.ZInfoMsg
		if err := jsonpb.Unmarshal(&buf, &emp); err != nil {
			return err
		}
		itemTimeStamp = emp.AtTimeStamp
	case int(MetricsType):
		pathToCheck = cacher.dirMetrics(devUUID)
		var emp metrics.ZMetricMsg
		if err := jsonpb.Unmarshal(&buf, &emp); err != nil {
			return err
		}
		itemTimeStamp = emp.AtTimeStamp
	default:
		return fmt.Errorf("not implemented type %d", typeToProcess)
	}
	if itemTimeStamp == nil {
		return fmt.Errorf("nil timestamp for data: %s", string(data))
	}
	pathToCheck = filepath.Join(pathToCheck, fmt.Sprintf("%d:%09d", itemTimeStamp.GetSeconds(), itemTimeStamp.GetNanos()))
	if err := os.MkdirAll(filepath.Dir(pathToCheck), 0755); err != nil {
		return err
	}
	if _, err := os.Stat(pathToCheck); os.IsNotExist(err) {
		return ioutil.WriteFile(pathToCheck, data, 0755)
	}
	return nil
}
