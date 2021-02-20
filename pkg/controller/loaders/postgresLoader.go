package loaders

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/lf-edge/eden/pkg/controller/cachers"
	"github.com/lf-edge/eden/pkg/controller/types"

	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
)

//PostgresLoader implements loader from postgres backend of controller
type PostgresLoader struct {
	lastID   string
	addr     string
	username string
	password string
	database string
	client   *pgxpool.Pool
	cache    cachers.CacheProcessor
	devUUID  uuid.UUID
	appUUID  uuid.UUID
}

//NewPostgresLoader return loader from postgres
func NewPostgresLoader(addr string, username string, password string, database string) *PostgresLoader {
	log.Debugf("NewPostgresLoader init")
	return &PostgresLoader{
		addr:     addr,
		username: username,
		password: password,
		database: database,
	}
}

//SetRemoteCache add cache layer
func (loader *PostgresLoader) SetRemoteCache(cache cachers.CacheProcessor) {
	loader.cache = cache
}

//Clone create copy
func (loader *PostgresLoader) Clone() Loader {
	return &PostgresLoader{
		addr:     loader.addr,
		username: loader.username,
		password: loader.password,
		database: loader.database,
		lastID:   "",
		cache:    loader.cache,
		devUUID:  loader.devUUID,
		appUUID:  loader.appUUID,
	}
}

//SetUUID set device UUID
func (loader *PostgresLoader) SetUUID(devUUID uuid.UUID) {
	loader.devUUID = devUUID
}

//SetAppUUID set app UUID
func (loader *PostgresLoader) SetAppUUID(appUUID uuid.UUID) {
	loader.appUUID = appUUID
}

func (loader *PostgresLoader) process(process ProcessFunction, typeToProcess types.LoaderObjectType, stream bool) error {
	tbl := ""
	id := loader.devUUID
	if typeToProcess == types.LogsType {
		tbl = "log"
	} else if typeToProcess == types.InfoType {
		tbl = "info"
	} else if typeToProcess == types.MetricsType {
		tbl = "metric"
	} else if typeToProcess == types.RequestType {
		tbl = "request"
	} else if typeToProcess == types.AppsType {
		tbl = "applog"
		id = loader.appUUID
	}
	if !stream {
		rows, err := loader.client.Query(context.Background(), fmt.Sprintf("SELECT data FROM %s WHERE ref = $1 ORDER BY id", tbl), id.String())
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var data []byte
			if err := rows.Scan(&data); err != nil {
				return err
			}
			tocontinue, err := process(data)
			if err != nil {
				return fmt.Errorf("process: %s", err)
			}
			if loader.cache != nil {
				if err = loader.cache.CheckAndSave(loader.devUUID, typeToProcess, data); err != nil {
					log.Errorf("error in cache: %s", err)
				}
			}
			if !tocontinue {
				return nil
			}
		}
		if err = rows.Err(); err != nil {
			return err
		}
	} else {
		conn, err := loader.client.Acquire(context.Background())
		if err != nil {
			return fmt.Errorf("error acquiring connection: %s", err)
		}
		defer conn.Release()
		_, err = conn.Exec(context.Background(), fmt.Sprintf("listen %s", tbl))
		if err != nil {
			return fmt.Errorf("error listening to chat channel: %s", err)
		}
		for {
			notification, err := conn.Conn().WaitForNotification(context.Background())
			if err != nil {
				return fmt.Errorf("error waiting for notification: %s", err)
			}
			var notificationData map[string]string
			if err := json.Unmarshal([]byte(notification.Payload), &notificationData); err != nil {
				return fmt.Errorf("error unmarshal of notification: %s", err)
			}
			if notificationData["ref"] != id.String() {
				continue
			}

			row := loader.client.QueryRow(context.Background(), fmt.Sprintf("SELECT data FROM %s where id = $1", tbl), notificationData["id"])
			var data []byte
			if err := row.Scan(&data); err != nil {
				return err
			}
			tocontinue, err := process(data)
			if err != nil {
				return fmt.Errorf("process: %s", err)
			}
			if loader.cache != nil {
				if err = loader.cache.CheckAndSave(loader.devUUID, typeToProcess, data); err != nil {
					log.Errorf("error in cache: %s", err)
				}
			}
			if !tocontinue {
				return nil
			}
		}
	}
	return nil
}

func (loader *PostgresLoader) repeatableConnection(process ProcessFunction, typeToProcess types.LoaderObjectType, stream bool) error {
	if err := loader.process(process, typeToProcess, stream); err != nil {
		log.Errorf("PostgresLoader repeatableConnection error: %s", err)
	}
	return nil
}

func (loader *PostgresLoader) getOrCreateClient() (*pgxpool.Pool, error) {
	if loader.client == nil {
		conn, err := pgxpool.Connect(context.Background(),
			fmt.Sprintf("postgres://%s:%s@%s/%s", loader.username, loader.password, loader.addr, loader.database))
		if err != nil {
			return nil, err
		}
		loader.client = conn
	}
	return loader.client, nil
}

//ProcessExisting for observe existing files
func (loader *PostgresLoader) ProcessExisting(process ProcessFunction, typeToProcess types.LoaderObjectType) error {
	if _, err := loader.getOrCreateClient(); err != nil {
		return err
	}
	return loader.repeatableConnection(process, typeToProcess, false)
}

//ProcessStream for observe new files
func (loader *PostgresLoader) ProcessStream(process ProcessFunction, typeToProcess types.LoaderObjectType, timeoutSeconds time.Duration) (err error) {
	if _, err := loader.getOrCreateClient(); err != nil {
		return err
	}
	done := make(chan error)
	if timeoutSeconds != 0 {
		time.AfterFunc(timeoutSeconds*time.Second, func() {
			done <- fmt.Errorf("timeout")
		})
	}

	go func() {
		done <- loader.repeatableConnection(process, typeToProcess, true)
	}()
	return <-done
}
