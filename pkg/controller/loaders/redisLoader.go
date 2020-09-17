package loaders

import (
	"fmt"
	"github.com/go-redis/redis/v7"
	"github.com/lf-edge/eden/pkg/controller/cachers"
	"github.com/lf-edge/eden/pkg/controller/types"
	"github.com/lf-edge/eden/pkg/defaults"
	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
	"strconv"
	"strings"
	"time"
)

type redisLoader struct {
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

//RedisLoader return loader from redis
func RedisLoader(addr string, password string, databaseID int, streamGetters types.StreamGetters) *redisLoader {
	log.Debugf("RedisLoader init")
	return &redisLoader{
		addr:          addr,
		password:      password,
		databaseID:    databaseID,
		streamGetters: streamGetters,
	}
}

//SetRemoteCache add cache layer
func (loader *redisLoader) SetRemoteCache(cache cachers.CacheProcessor) {
	loader.cache = cache
}

//Clone create copy
func (loader *redisLoader) Clone() Loader {
	return &redisLoader{
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

func (loader *redisLoader) getStream(typeToProcess types.LoaderObjectType) string {
	switch typeToProcess {
	case types.LogsType:
		return loader.streamGetters.StreamLogs(loader.devUUID)
	case types.InfoType:
		return loader.streamGetters.StreamInfo(loader.devUUID)
	case types.MetricsType:
		return loader.streamGetters.StreamMetrics(loader.devUUID)
	case types.RequestType:
		return loader.streamGetters.StreamRequest(loader.devUUID)
	case types.AppsType:
		return loader.streamGetters.StreamApps(loader.devUUID, loader.appUUID)
	default:
		return ""
	}
}

//SetUUID set device UUID
func (loader *redisLoader) SetUUID(devUUID uuid.UUID) {
	loader.devUUID = devUUID
}

//SetUUID set app UUID
func (loader *redisLoader) SetAppUUID(appUUID uuid.UUID) {
	loader.appUUID = appUUID
}

func (loader *redisLoader) process(process ProcessFunction, typeToProcess types.LoaderObjectType, stream bool) (processed, found bool, err error) {
	OrderStream := loader.getStream(typeToProcess)
	if !stream {
		start := "-"
		for {
			rr, err := loader.client.XRangeN(OrderStream, start, "+", 10).Result()
			if err != nil {
				return false, false, fmt.Errorf("XRange error: %s", err)
			}

			if len(rr) == 0 {
				return true, false, nil
			}

			for _, r := range rr {
				loader.lastID = r.ID
				log.Debugf("lastID: %s", loader.lastID)
				data := []byte(r.Values["object"].(string))
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
		log.Debugf("XRead from %s", OrderStream)
		start := "$"
		rr, err := loader.client.XRead(&redis.XReadArgs{
			Streams: []string{OrderStream, start},
			Count:   1,
			Block:   0,
		}).Result()
		if err != nil {
			return false, false, fmt.Errorf("XRead error: %s", err)
		}

		for _, r := range rr[0].Messages {
			loader.lastID = r.ID
			log.Debugf("XRead lastID: %s", loader.lastID)
			data := []byte(r.Values["object"].(string))
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
			rr, err := loader.client.XRangeN(OrderStream, start, "+", 100).Result()
			if err != nil {
				return false, false, fmt.Errorf("XRange error: %s", err)
			}

			for _, r := range rr {
				loader.lastID = r.ID
				log.Debugf("XRangeN lastID: %s", loader.lastID)
				data := []byte(r.Values["object"].(string))
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

func (loader *redisLoader) repeatableConnection(process ProcessFunction, typeToProcess types.LoaderObjectType, stream bool) error {
	if _, _, err := loader.process(process, typeToProcess, stream); err == nil {
		return nil
	} else {
		log.Errorf("redisLoader repeatableConnection error: %s", err)
	}
	return fmt.Errorf("all connection attempts failed")
}

func (loader *redisLoader) getOrCreateClient() (*redis.Client, error) {
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
	_, err := loader.client.Ping().Result()
	return loader.client, err
}

//ProcessExisting for observe existing files
func (loader *redisLoader) ProcessExisting(process ProcessFunction, typeToProcess types.LoaderObjectType) error {
	if _, err := loader.getOrCreateClient(); err != nil {
		return err
	}
	return loader.repeatableConnection(process, typeToProcess, false)
}

//ProcessExisting for observe new files
func (loader *redisLoader) ProcessStream(process ProcessFunction, typeToProcess types.LoaderObjectType, timeoutSeconds time.Duration) (err error) {
	if _, err := loader.getOrCreateClient(); err != nil {
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
		done <- loader.repeatableConnection(process, typeToProcess, true)
	}()
	if err = <-done; err != nil {
		return err
	}
	return loader.client.Close()
}
