package loaders

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis/v9"
	"github.com/lf-edge/eden/pkg/controller/cachers"
	"github.com/lf-edge/eden/pkg/controller/types"
	"github.com/lf-edge/eden/pkg/defaults"
	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
)

// RedisLoader implements loader from redis backend of controller
type RedisLoader struct {
	lastID        string
	addr          string
	password      string
	databaseID    int
	streamGetters types.StreamGetters
	client        *redis.Client
	cache         cachers.CacheProcessor
	devUUID       uuid.UUID
	appUUID       uuid.UUID
}

// NewRedisLoader return loader from redis
func NewRedisLoader(addr string, password string, databaseID int, streamGetters types.StreamGetters) *RedisLoader {
	log.Debugf("NewRedisLoader init")
	return &RedisLoader{
		addr:          addr,
		password:      password,
		databaseID:    databaseID,
		streamGetters: streamGetters,
	}
}

// SetRemoteCache add cache layer
func (loader *RedisLoader) SetRemoteCache(cache cachers.CacheProcessor) {
	loader.cache = cache
}

// Clone create copy
func (loader *RedisLoader) Clone() Loader {
	return &RedisLoader{
		addr:          loader.addr,
		password:      loader.password,
		databaseID:    loader.databaseID,
		streamGetters: loader.streamGetters,
		lastID:        "",
		cache:         loader.cache,
		devUUID:       loader.devUUID,
		appUUID:       loader.appUUID,
	}
}

func (loader *RedisLoader) getStream(typeToProcess types.LoaderObjectType) string {
	switch typeToProcess {
	case types.LogsType:
		return loader.streamGetters.StreamLogs(loader.devUUID)
	case types.InfoType:
		return loader.streamGetters.StreamInfo(loader.devUUID)
	case types.MetricsType:
		return loader.streamGetters.StreamMetrics(loader.devUUID)
	case types.FlowLogType:
		return loader.streamGetters.StreamFlowLog(loader.devUUID)
	case types.RequestType:
		return loader.streamGetters.StreamRequest(loader.devUUID)
	case types.AppsType:
		return loader.streamGetters.StreamApps(loader.devUUID, loader.appUUID)
	default:
		return ""
	}
}

// SetUUID set device UUID
func (loader *RedisLoader) SetUUID(devUUID uuid.UUID) {
	loader.devUUID = devUUID
}

// SetAppUUID set app UUID
func (loader *RedisLoader) SetAppUUID(appUUID uuid.UUID) {
	loader.appUUID = appUUID
}

func (loader *RedisLoader) process(process ProcessFunction, typeToProcess types.LoaderObjectType, stream bool) (processed, found bool, err error) {
	OrderStream := loader.getStream(typeToProcess)
	log.Debugf("XRead from %s", OrderStream)
	if !stream {
		start := "-"
		for {
			rr, err := loader.client.XRangeN(context.Background(), OrderStream, start, "+", 10).Result()
			if err != nil {
				return false, false, fmt.Errorf("XRange error: %s", err)
			}

			if len(rr) == 0 {
				return true, false, nil
			}

			for _, r := range rr {
				loader.lastID = r.ID
				dataString, ok := r.Values["object"].(string)
				if !ok {
					continue
				}
				data := []byte(dataString)
				tocontinue, err := process(data)
				if err != nil {
					return false, false, fmt.Errorf("process: %s", err)
				}
				if loader.cache != nil {
					if err = loader.cache.CheckAndSave(loader.devUUID, typeToProcess, data); err != nil {
						log.Errorf("error in cache: %s", err)
					}
				}
				if !tocontinue {
					return true, true, nil
				}
			}
			splitted := strings.Split(loader.lastID, "-")
			counter, _ := strconv.Atoi(splitted[1])
			start = fmt.Sprintf("%s-%v", splitted[0], counter+1)
		}
	} else {
		start := "$"
		rr, err := loader.client.XRead(context.Background(), &redis.XReadArgs{
			Streams: []string{OrderStream, start},
			Count:   1,
			Block:   0,
		}).Result()
		if err != nil {
			return false, false, fmt.Errorf("XRead error: %s", err)
		}

		for _, r := range rr[0].Messages {
			loader.lastID = r.ID
			dataString, ok := r.Values["object"].(string)
			if !ok {
				continue
			}
			data := []byte(dataString)
			tocontinue, err := process(data)
			if err != nil {
				return false, false, fmt.Errorf("process first: %s", err)
			}
			if loader.cache != nil {
				if err = loader.cache.CheckAndSave(loader.devUUID, typeToProcess, data); err != nil {
					log.Errorf("error in cache: %s", err)
				}
			}
			if !tocontinue {
				return true, true, nil
			}
		}
		splitted := strings.Split(loader.lastID, "-")
		counter, _ := strconv.Atoi(splitted[1])
		start = fmt.Sprintf("%s-%v", splitted[0], counter+1)
		for {
			rr, err := loader.client.XRangeN(context.Background(), OrderStream, start, "+", 100).Result()
			if err != nil {
				return false, false, fmt.Errorf("XRange error: %s", err)
			}

			for _, r := range rr {
				loader.lastID = r.ID
				dataString, ok := r.Values["object"].(string)
				if !ok {
					continue
				}
				data := []byte(dataString)
				tocontinue, err := process(data)
				if err != nil {
					return false, false, fmt.Errorf("process: %s", err)
				}
				if loader.cache != nil {
					if err = loader.cache.CheckAndSave(loader.devUUID, typeToProcess, data); err != nil {
						log.Errorf("error in cache: %s", err)
					}
				}
				if !tocontinue {
					return true, true, nil
				}
			}
			splitted := strings.Split(loader.lastID, "-")
			counter, _ := strconv.Atoi(splitted[1])
			start = fmt.Sprintf("%s-%v", splitted[0], counter+1)
			time.Sleep(time.Second) //sleep for second to reduce the load of redis
		}
	}
}

func (loader *RedisLoader) repeatableConnection(process ProcessFunction, typeToProcess types.LoaderObjectType, stream bool) error {
	if _, _, err := loader.process(process, typeToProcess, stream); err != nil {
		log.Errorf("RedisLoader repeatableConnection error: %s", err)
	}
	return nil
}

func (loader *RedisLoader) getOrCreateClient() (*redis.Client, error) {
	if loader.client == nil {
		loader.client = redis.NewClient(&redis.Options{
			Addr:            loader.addr,
			Password:        loader.password,
			DB:              loader.databaseID,
			MaxRetries:      defaults.DefaultRepeatCount,
			MinRetryBackoff: defaults.DefaultRepeatTimeout / 2,
			MaxRetryBackoff: defaults.DefaultRepeatTimeout * 2,
		})
	}
	_, err := loader.client.Ping(context.Background()).Result()
	return loader.client, err
}

// ProcessExisting for observe existing files
func (loader *RedisLoader) ProcessExisting(process ProcessFunction, typeToProcess types.LoaderObjectType) error {
	if _, err := loader.getOrCreateClient(); err != nil {
		return err
	}
	return loader.repeatableConnection(process, typeToProcess, false)
}

// ProcessStream for observe new files
func (loader *RedisLoader) ProcessStream(process ProcessFunction, typeToProcess types.LoaderObjectType, timeoutSeconds time.Duration) (err error) {
	if _, err := loader.getOrCreateClient(); err != nil {
		return err
	}
	done := make(chan error)
	if timeoutSeconds != 0 {
		time.AfterFunc(timeoutSeconds, func() {
			done <- fmt.Errorf("timeout")
		})
	}

	go func() {
		done <- loader.repeatableConnection(process, typeToProcess, true)
	}()
	if err = <-done; err != nil {
		return err
	}
	return loader.client.Close()
}
