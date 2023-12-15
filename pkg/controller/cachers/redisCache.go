package cachers

import (
	"bytes"
	"context"
	"fmt"

	"github.com/go-redis/redis/v9"
	"github.com/lf-edge/eden/pkg/controller/types"
	"github.com/lf-edge/eve-api/go/info"
	"github.com/lf-edge/eve-api/go/logs"
	"github.com/lf-edge/eve-api/go/metrics"
	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// RedisCache object provides caching objects from controller into redis
type RedisCache struct {
	addr          string
	password      string
	databaseID    int
	streamGetters types.StreamGetters
	client        *redis.Client
}

// NewRedisCache creates new RedisCache with provided settings
func NewRedisCache(addr string, password string, databaseID int, streamGetters types.StreamGetters) *RedisCache {
	return &RedisCache{
		addr:          addr,
		password:      password,
		databaseID:    databaseID,
		streamGetters: streamGetters,
	}
}

func (cacher *RedisCache) newRedisClient() (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cacher.addr,
		Password: cacher.password,
		DB:       cacher.databaseID,
	})
	_, err := client.Ping(context.Background()).Result()
	return client, err
}

// CheckAndSave process LoaderObjectType from data
func (cacher *RedisCache) CheckAndSave(devUUID uuid.UUID, typeToProcess types.LoaderObjectType, data []byte) (err error) {
	if cacher.client == nil {
		if cacher.client, err = cacher.newRedisClient(); err != nil {
			return err
		}
	}

	var streamToWrite string
	var itemTimeStamp *timestamppb.Timestamp
	var buf bytes.Buffer
	buf.Write(data)
	switch typeToProcess {
	case types.LogsType:
		streamToWrite = cacher.streamGetters.StreamLogs(devUUID)
		var emp logs.LogBundle
		if err := protojson.Unmarshal(buf.Bytes(), &emp); err != nil {
			return err
		}
		itemTimeStamp = emp.Timestamp
	case types.InfoType:
		streamToWrite = cacher.streamGetters.StreamInfo(devUUID)
		var emp info.ZInfoMsg
		if err := protojson.Unmarshal(buf.Bytes(), &emp); err != nil {
			return err
		}
		itemTimeStamp = emp.AtTimeStamp
	case types.MetricsType:
		streamToWrite = cacher.streamGetters.StreamMetrics(devUUID)
		var emp metrics.ZMetricMsg
		if err := protojson.Unmarshal(buf.Bytes(), &emp); err != nil {
			return err
		}
		itemTimeStamp = emp.AtTimeStamp
	default:
		return fmt.Errorf("not implemented type %d", typeToProcess)
	}
	rr, err := cacher.client.XRange(context.Background(), streamToWrite, "-", "+").Result()
	if err != nil {
		return err
	}
	for _, r := range rr {
		switch typeToProcess {
		case types.LogsType:
			var buf bytes.Buffer
			buf.Write([]byte(r.Values["object"].(string)))
			var emp logs.LogBundle
			if err := protojson.Unmarshal(buf.Bytes(), &emp); err != nil {
				return err
			}
			if emp.Timestamp.GetSeconds() == itemTimeStamp.GetSeconds() && emp.Timestamp.GetNanos() == itemTimeStamp.GetNanos() {
				return
			}
		case types.InfoType:
			var buf bytes.Buffer
			buf.Write([]byte(r.Values["object"].(string)))
			var emp info.ZInfoMsg
			if err := protojson.Unmarshal(buf.Bytes(), &emp); err != nil {
				return err
			}
			if emp.AtTimeStamp.GetSeconds() == itemTimeStamp.GetSeconds() && emp.AtTimeStamp.GetNanos() == itemTimeStamp.GetNanos() {
				return
			}
		default:
			return fmt.Errorf("not implemented type %d", typeToProcess)
		}
	}

	strCMD := cacher.client.XAdd(context.Background(), &redis.XAddArgs{
		Stream: streamToWrite,
		Values: map[string]interface{}{
			"object": data,
		},
	})
	var key string
	if key, err = strCMD.Result(); err != nil {
		return fmt.Errorf("error in XAdd:%v", err)
	}
	log.Debugf("ready with write to redis %s: %s", key, data)
	return nil
}
