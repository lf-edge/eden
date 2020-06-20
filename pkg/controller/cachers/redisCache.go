package cachers

import (
	"bytes"
	"fmt"
	"github.com/go-redis/redis/v7"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/lf-edge/eve/api/go/info"
	"github.com/lf-edge/eve/api/go/logs"
	"github.com/lf-edge/eve/api/go/metrics"
	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
)

type getStream = func(devUUID uuid.UUID) (stream string)

type redisCache struct {
	addr          string
	password      string
	databaseID    int
	streamLogs    getStream
	streamInfo    getStream
	streamMetrics getStream
	client        *redis.Client
}

func RedisCache(addr string, password string, databaseID int, streamLogs getStream, streamInfo getStream, streamMetrics getStream) *redisCache {
	return &redisCache{
		addr:          addr,
		password:      password,
		databaseID:    databaseID,
		streamLogs:    streamLogs,
		streamInfo:    streamInfo,
		streamMetrics: streamMetrics,
	}
}

func (cacher *redisCache) newRedisClient() (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cacher.addr,
		Password: cacher.password,
		DB:       cacher.databaseID,
	})
	_, err := client.Ping().Result()
	return client, err
}

func (cacher *redisCache) CheckAndSave(devUUID uuid.UUID, typeToProcess int, data []byte) (err error) {
	if cacher.client == nil {
		if cacher.client, err = cacher.newRedisClient(); err != nil {
			return err
		}
	}

	var streamToWrite string
	var itemTimeStamp *timestamp.Timestamp
	var buf bytes.Buffer
	buf.Write(data)
	switch typeToProcess {
	case int(LogsType):
		streamToWrite = cacher.streamLogs(devUUID)
		var emp logs.LogBundle
		if err := jsonpb.Unmarshal(&buf, &emp); err != nil {
			return err
		}
		itemTimeStamp = emp.Timestamp
	case int(InfoType):
		streamToWrite = cacher.streamInfo(devUUID)
		var emp info.ZInfoMsg
		if err := jsonpb.Unmarshal(&buf, &emp); err != nil {
			return err
		}
		itemTimeStamp = emp.AtTimeStamp
	case int(MetricsType):
		streamToWrite = cacher.streamMetrics(devUUID)
		var emp metrics.ZMetricMsg
		if err := jsonpb.Unmarshal(&buf, &emp); err != nil {
			return err
		}
		itemTimeStamp = emp.AtTimeStamp
	default:
		return fmt.Errorf("not implemented type %d", typeToProcess)
	}
	rr, err := cacher.client.XRange(streamToWrite, "-", "+").Result()
	if err != nil {
		return err
	}
	for _, r := range rr {
		switch typeToProcess {
		case int(LogsType):
			var buf bytes.Buffer
			buf.Write([]byte(r.Values["object"].(string)))
			var emp logs.LogBundle
			if err := jsonpb.Unmarshal(&buf, &emp); err != nil {
				return err
			}
			if emp.Timestamp.GetSeconds() == itemTimeStamp.GetSeconds() && emp.Timestamp.GetNanos() == itemTimeStamp.GetNanos() {
				return
			}
		case int(InfoType):
			var buf bytes.Buffer
			buf.Write([]byte(r.Values["object"].(string)))
			var emp info.ZInfoMsg
			if err := jsonpb.Unmarshal(&buf, &emp); err != nil {
				return err
			}
			if emp.AtTimeStamp.GetSeconds() == itemTimeStamp.GetSeconds() && emp.AtTimeStamp.GetNanos() == itemTimeStamp.GetNanos() {
				return
			}
		default:
			return fmt.Errorf("not implemented type %d", typeToProcess)
		}
	}

	strCMD := cacher.client.XAdd(&redis.XAddArgs{
		Stream: streamToWrite,
		Values: map[string]interface{}{
			"object": data,
		},
	})
	var key string
	if key, err = strCMD.Result(); err != nil {
		return fmt.Errorf("XAdd error:%v\n", err)
	}
	log.Debugf("ready with write to redis %s: %s", key, data)
	return nil
}
