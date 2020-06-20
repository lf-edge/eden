package adam

import (
	"encoding/json"
	"fmt"
	"github.com/lf-edge/adam/pkg/server"
	"github.com/lf-edge/adam/pkg/x509"
	"github.com/lf-edge/eden/pkg/controller/cachers"
	"github.com/lf-edge/eden/pkg/controller/einfo"
	"github.com/lf-edge/eden/pkg/controller/elog"
	"github.com/lf-edge/eden/pkg/controller/loaders"
	"github.com/lf-edge/eden/pkg/controller/types"
	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/device"
	"github.com/lf-edge/eden/pkg/utils"
	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"time"
)

type Ctx struct {
	dir               string
	url               string
	serverCA          string
	insecureTLS       bool
	AdamRemote        bool
	AdamRemoteRedis   bool   //use redis for obtain logs and info
	AdamRedisUrlEden  string //string with redis url for obtain logs and info
	AdamCaching       bool   //enable caching of adam`s logs/info
	AdamCachingRedis  bool   //caching to redis instead of files
	AdamCachingPrefix string //custom prefix for file or stream naming for cache
}

//parseRedisUrl try to use string from config to obtain redis url
func parseRedisUrl(s string) (addr, password string, databaseID int, err error) {
	URL, err := url.Parse(s)
	if err != nil || URL.Scheme != "redis" {
		return "", "", 0, err
	}

	if URL.Host != "" {
		addr = URL.Host
	} else {
		addr = fmt.Sprintf("%s:%s", defaults.DefaultRedisHost, defaults.DefaultRedisPort)
	}
	if URL.Path != "" {
		if databaseID, err = strconv.Atoi(strings.Trim(URL.Path, "/")); err != nil {
			return "", "", 0, err
		}
	} else {
		databaseID = 0
	}
	password = URL.User.Username()
	return
}

//getLoader return loader object from Adam`s config
func (adam *Ctx) getLoader() (loader loaders.Loader) {
	if adam.AdamRemote {
		log.Debug("will use remote adam loader")
		if adam.AdamRemoteRedis {
			addr, password, databaseID, err := parseRedisUrl(adam.AdamRedisUrlEden)
			if err != nil {
				log.Fatalf("Cannot parse adam redis url: %s", err)
			}
			loader = loaders.RedisLoader(addr, password, databaseID, adam.getLogsRedisStream, adam.getInfoRedisStream, adam.getMetricsRedisStream)
		} else {
			loader = loaders.RemoteLoader(adam.getHTTPClient, adam.getLogsUrl, adam.getInfoUrl, adam.getMetricsUrl)
		}
	} else {
		log.Debug("will use local adam loader")
		loader = loaders.FileLoader(adam.getLogsDir, adam.getInfoDir, adam.getMetricsDir)
	}
	if adam.AdamCaching {
		var cache cachers.Cacher
		if adam.AdamCachingRedis {
			addr, password, databaseID, err := parseRedisUrl(adam.AdamRedisUrlEden)
			if err != nil {
				log.Fatalf("Cannot parse adam redis url: %s", err)
			}
			cache = cachers.RedisCache(addr, password, databaseID, adam.getLogsRedisStreamCache, adam.getInfoRedisStreamCache, adam.getMetricsRedisStreamCache)
		} else {
			cache = cachers.FileCache(adam.getLogsDirCache, adam.getInfoDirCache, adam.getMetricsDirCache)
		}
		loader.SetRemoteCache(cache)
	}
	return
}

//EnvRead use variables from viper for init controller
func (adam *Ctx) InitWithVars(vars *utils.ConfigVars) error {
	adam.dir = vars.AdamDir
	adam.url = fmt.Sprintf("https://%s:%s", vars.AdamIP, vars.AdamPort)
	adam.insecureTLS = len(vars.AdamCA) == 0
	adam.serverCA = vars.AdamCA
	adam.AdamRemote = vars.AdamRemote
	adam.AdamRemoteRedis = vars.AdamRemoteRedis
	adam.AdamCaching = vars.AdamCaching
	adam.AdamCachingRedis = vars.AdamCachingRedis
	adam.AdamCachingPrefix = vars.AdamCachingPrefix
	adam.AdamRedisUrlEden = vars.AdamRedisUrlEden
	return nil
}

//GetDir return dir
func (adam *Ctx) GetDir() (dir string) {
	return adam.dir
}

//getLogsRedisStream return info stream for devUUID for load from redis
func (adam *Ctx) getLogsRedisStream(devUUID uuid.UUID) (dir string) {
	return fmt.Sprintf("%s%s", defaults.DefaultLogsRedisPrefix, devUUID.String())
}

//getInfoRedisStream return info stream for devUUID for load from redis
func (adam *Ctx) getInfoRedisStream(devUUID uuid.UUID) (dir string) {
	return fmt.Sprintf("%s%s", defaults.DefaultInfoRedisPrefix, devUUID.String())
}

//getMetricsRedisStream return metrics stream for devUUID for load from redis
func (adam *Ctx) getMetricsRedisStream(devUUID uuid.UUID) (dir string) {
	return fmt.Sprintf("%s%s", defaults.DefaultMetricsRedisPrefix, devUUID.String())
}

//getLogsRedisStreamCache return logs stream for devUUID for caching in redis
func (adam *Ctx) getLogsRedisStreamCache(devUUID uuid.UUID) (dir string) {
	if adam.AdamCachingPrefix == "" {
		return adam.getLogsRedisStream(devUUID)
	}
	return fmt.Sprintf("LOGS_EVE_%s_%s", adam.AdamCachingPrefix, devUUID.String())
}

//getInfoRedisStreamCache return info stream for devUUID for caching in redis
func (adam *Ctx) getInfoRedisStreamCache(devUUID uuid.UUID) (dir string) {
	if adam.AdamCachingPrefix == "" {
		return adam.getInfoRedisStream(devUUID)
	}
	return fmt.Sprintf("INFO_EVE_%s_%s", adam.AdamCachingPrefix, devUUID.String())
}

//getMetricsRedisStreamCache return metrics stream for devUUID for caching in redis
func (adam *Ctx) getMetricsRedisStreamCache(devUUID uuid.UUID) (dir string) {
	if adam.AdamCachingPrefix == "" {
		return adam.getMetricsRedisStream(devUUID)
	}
	return fmt.Sprintf("METRICS_EVE_%s_%s", adam.AdamCachingPrefix, devUUID.String())
}

//getRedisStreamCache return logs stream for devUUID for caching in redis
func (adam *Ctx) getLogsDirCache(devUUID uuid.UUID) (dir string) {
	if adam.AdamCachingPrefix == "" {
		return adam.getLogsDir(devUUID)
	}
	return path.Join(adam.dir, adam.AdamCachingPrefix, devUUID.String(), "logs")
}

//getInfoDirCache return info directory for devUUID for caching
func (adam *Ctx) getInfoDirCache(devUUID uuid.UUID) (dir string) {
	if adam.AdamCachingPrefix == "" {
		return adam.getInfoDir(devUUID)
	}
	return path.Join(adam.dir, adam.AdamCachingPrefix, devUUID.String(), "info")
}

//getMetricsDirCache return metrics directory for devUUID for caching
func (adam *Ctx) getMetricsDirCache(devUUID uuid.UUID) (dir string) {
	if adam.AdamCachingPrefix == "" {
		return adam.getMetricsDir(devUUID)
	}
	return path.Join(adam.dir, adam.AdamCachingPrefix, devUUID.String(), "metrics")
}

//getLogsDir return logs directory for devUUID
func (adam *Ctx) getLogsDir(devUUID uuid.UUID) (dir string) {
	return path.Join(adam.dir, "run", "adam", "device", devUUID.String(), "logs")
}

//getInfoDir return info directory for devUUID
func (adam *Ctx) getInfoDir(devUUID uuid.UUID) (dir string) {
	return path.Join(adam.dir, "run", "adam", "device", devUUID.String(), "info")
}

//getMetricsDir return metrics directory for devUUID
func (adam *Ctx) getMetricsDir(devUUID uuid.UUID) (dir string) {
	return path.Join(adam.dir, "run", "adam", "device", devUUID.String(), "metrics")
}

//getLogsUrl return logs url for devUUID
func (adam *Ctx) getLogsUrl(devUUID uuid.UUID) string {
	resUrl, err := utils.ResolveURL(adam.url, path.Join("/admin/device", devUUID.String(), "logs"))
	if err != nil {
		log.Fatalf("ResolveURL: %s", err)
	}
	return resUrl
}

//getLogsUrl return info url for devUUID
func (adam *Ctx) getInfoUrl(devUUID uuid.UUID) string {
	resUrl, err := utils.ResolveURL(adam.url, path.Join("/admin/device", devUUID.String(), "info"))
	if err != nil {
		log.Fatalf("ResolveURL: %s", err)
	}
	return resUrl
}

//getMetricsUrl return metrics url for devUUID
func (adam *Ctx) getMetricsUrl(devUUID uuid.UUID) string {
	resUrl, err := utils.ResolveURL(adam.url, path.Join("/admin/device", devUUID.String(), "metrics"))
	if err != nil {
		log.Fatalf("ResolveURL: %s", err)
	}
	return resUrl
}

//Register device in adam
func (adam *Ctx) Register(device *device.Ctx) error {
	b, err := ioutil.ReadFile(device.GetOnboardKey())
	switch {
	case err != nil && os.IsNotExist(err):
		log.Printf("cert file %s does not exist", device.GetOnboardKey())
		return err
	case err != nil:
		log.Printf("error reading cert file %s: %v", device.GetOnboardKey(), err)
		return err
	}

	objToSend := server.OnboardCert{
		Cert:   b,
		Serial: device.GetSerial(),
	}
	body, err := json.Marshal(objToSend)
	if err != nil {
		log.Printf("error encoding json: %v", err)
		return err
	}
	return adam.postObj("/admin/onboard", body)
}

//DeviceList return device list
func (adam *Ctx) DeviceList(filter types.DeviceStateFilter) (out []string, err error) {
	if filter == types.RegisteredDeviceFilter || filter == types.AllDevicesFilter {
		return adam.getList("/admin/device")
	} else {
		return []string{}, nil
	}
}

//ConfigSet set config for devID
func (adam *Ctx) ConfigSet(devUUID uuid.UUID, devConfig []byte) (err error) {
	return adam.putObj(path.Join("/admin/device", devUUID.String(), "config"), devConfig)
}

//ConfigGet get config for devID
func (adam *Ctx) ConfigGet(devUUID uuid.UUID) (out string, err error) {
	return adam.getObj(path.Join("/admin/device", devUUID.String(), "config"))
}

//LogChecker check logs by pattern from existence files with LogLast and use LogWatchWithTimeout with timeout for observe new files
func (adam *Ctx) LogChecker(devUUID uuid.UUID, q map[string]string, handler elog.HandlerFunc, mode elog.LogCheckerMode, timeout time.Duration) (err error) {
	return elog.LogChecker(adam.getLoader(), devUUID, q, handler, mode, timeout)
}

//LogLastCallback check logs by pattern from existence files with callback
func (adam *Ctx) LogLastCallback(devUUID uuid.UUID, q map[string]string, handler elog.HandlerFunc) (err error) {
	var loader = adam.getLoader()
	loader.SetUUID(devUUID)
	return elog.LogLast(loader, q, handler)
}

//InfoChecker checks the information in the regular expression pattern 'query' and processes the info.ZInfoMsg found by the function 'handler' from existing files (mode=einfo.InfoExist), new files (mode=einfo.InfoNew) or any of them (mode=einfo.InfoAny) with timeout.
func (adam *Ctx) InfoChecker(devUUID uuid.UUID, q map[string]string, infoType einfo.ZInfoType, handler einfo.HandlerFunc, mode einfo.InfoCheckerMode, timeout time.Duration) (err error) {
	return einfo.InfoChecker(adam.getLoader(), devUUID, q, infoType, handler, mode, timeout)
}

//InfoLastCallback check info by pattern from existence files with callback
func (adam *Ctx) InfoLastCallback(devUUID uuid.UUID, q map[string]string, infoType einfo.ZInfoType, handler einfo.HandlerFunc) (err error) {
	var loader = adam.getLoader()
	loader.SetUUID(devUUID)
	return einfo.InfoLast(loader, q, einfo.ZInfoFind, handler, infoType)
}

//DeviceGetOnboard get device onboardUUID for devUUID
func (adam *Ctx) DeviceGetOnboard(devUUID uuid.UUID) (onboardUUID uuid.UUID, err error) {
	var devCert server.DeviceCert
	devInfo, err := adam.getObj(path.Join("/admin/device", devUUID.String()))
	if err != nil {
		return uuid.Nil, err
	}
	if err = json.Unmarshal([]byte(devInfo), &devCert); err != nil {
		return uuid.Nil, err
	}

	cert, err := x509.ParseCert(devCert.Onboard)
	if err != nil {
		return uuid.Nil, err
	}
	return uuid.FromString(cert.Subject.CommonName)
}

//DeviceGetByOnboard try to get device by onboard eveCert
func (adam *Ctx) DeviceGetByOnboard(eveCert string) (devUUID uuid.UUID, err error) {
	b, err := ioutil.ReadFile(eveCert)
	switch {
	case err != nil && os.IsNotExist(err):
		log.Printf("cert file %s does not exist", eveCert)
		return uuid.Nil, err
	case err != nil:
		log.Printf("error reading cert file %s: %v", eveCert, err)
		return uuid.Nil, err
	}
	cert, err := x509.ParseCert(b)
	if err != nil {
		return uuid.Nil, err
	}
	uuidToFound, err := uuid.FromString(cert.Subject.CommonName)
	if err != nil {
		return uuid.Nil, err
	}
	devIDs, err := adam.DeviceList(types.RegisteredDeviceFilter)
	if err != nil {
		return uuid.Nil, err
	}
	for _, devID := range devIDs {
		devUUID, err := uuid.FromString(devID)
		if err != nil {
			return uuid.Nil, err
		}
		if onboardUUID, err := adam.DeviceGetOnboard(devUUID); err == nil {
			if onboardUUID.String() == uuidToFound.String() {
				return devUUID, nil
			}
		} else {
			return uuid.Nil, err
		}
	}
	return uuid.Nil, fmt.Errorf("no device found")
}
