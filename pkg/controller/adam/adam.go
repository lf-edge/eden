package adam

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/lf-edge/adam/pkg/server"
	"github.com/lf-edge/adam/pkg/x509"
	"github.com/lf-edge/eden/pkg/controller/cachers"
	"github.com/lf-edge/eden/pkg/controller/einfo"
	"github.com/lf-edge/eden/pkg/controller/elog"
	"github.com/lf-edge/eden/pkg/controller/emetric"
	"github.com/lf-edge/eden/pkg/controller/erequest"
	"github.com/lf-edge/eden/pkg/controller/loaders"
	"github.com/lf-edge/eden/pkg/controller/types"
	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/device"
	"github.com/lf-edge/eden/pkg/utils"
	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
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
			streamGetters := types.StreamGetters{
				StreamLogs:    adam.getLogsRedisStream,
				StreamInfo:    adam.getInfoRedisStream,
				StreamMetrics: adam.getMetricsRedisStream,
				StreamRequest: adam.getRequestRedisStream,
			}
			loader = loaders.RedisLoader(addr, password, databaseID, streamGetters)
		} else {
			urlGetters := types.UrlGetters{
				UrlLogs:    adam.getLogsUrl,
				UrlInfo:    adam.getInfoUrl,
				UrlMetrics: adam.getMetricsUrl,
				UrlRequest: adam.getRequestUrl,
			}
			loader = loaders.RemoteLoader(adam.getHTTPClient, urlGetters)
		}
	} else {
		log.Debug("will use local adam loader")
		dirGetters := types.DirGetters{
			LogsGetter:    adam.getLogsDir,
			InfoGetter:    adam.getInfoDir,
			MetricsGetter: adam.getMetricsDir,
			RequestGetter: adam.getRequestDir,
		}
		loader = loaders.FileLoader(dirGetters)
	}
	if adam.AdamCaching {
		var cache cachers.CacheProcessor
		if adam.AdamCachingRedis {
			addr, password, databaseID, err := parseRedisUrl(adam.AdamRedisUrlEden)
			if err != nil {
				log.Fatalf("Cannot parse adam redis url: %s", err)
			}
			streamGetters := types.StreamGetters{
				StreamLogs:    adam.getLogsRedisStreamCache,
				StreamInfo:    adam.getInfoRedisStreamCache,
				StreamMetrics: adam.getMetricsRedisStreamCache,
				StreamRequest: adam.getRequestRedisStreamCache,
			}
			cache = cachers.RedisCache(addr, password, databaseID, streamGetters)
		} else {
			dirGetters := types.DirGetters{
				LogsGetter:    adam.getLogsDirCache,
				InfoGetter:    adam.getInfoDirCache,
				MetricsGetter: adam.getMetricsDirCache,
				RequestGetter: adam.getRequestDirCache,
			}
			cache = cachers.FileCache(dirGetters)
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

//RequestLastCallback check request by pattern from existence files with callback
func (adam *Ctx) RequestLastCallback(devUUID uuid.UUID, q map[string]string, handler erequest.HandlerFunc) (err error) {
	var loader = adam.getLoader()
	loader.SetUUID(devUUID)
	return erequest.RequestLast(loader, q, handler)
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
func (adam *Ctx) InfoChecker(devUUID uuid.UUID, q map[string]string, handler einfo.HandlerFunc, mode einfo.InfoCheckerMode, timeout time.Duration) (err error) {
	return einfo.InfoChecker(adam.getLoader(), devUUID, q, handler, mode, timeout)
}

//InfoLastCallback check info by pattern from existence files with callback
func (adam *Ctx) InfoLastCallback(devUUID uuid.UUID, q map[string]string, handler einfo.HandlerFunc) (err error) {
	var loader = adam.getLoader()
	loader.SetUUID(devUUID)
	return einfo.InfoLast(loader, q, einfo.ZInfoFind, handler)
}

//MetricChecker check metrics by pattern from existence files with LogLast and use LogWatchWithTimeout with timeout for observe new files
func (adam *Ctx) MetricChecker(devUUID uuid.UUID, q map[string]string, handler emetric.HandlerFunc, mode emetric.MetricCheckerMode, timeout time.Duration) (err error) {
	return emetric.MetricChecker(adam.getLoader(), devUUID, q, handler, mode, timeout)
}

//MetricLastCallback check metrics by pattern from existence files with callback
func (adam *Ctx) MetricLastCallback(devUUID uuid.UUID, q map[string]string, handler emetric.HandlerFunc) (err error) {
	var loader = adam.getLoader()
	loader.SetUUID(devUUID)
	return emetric.MetricLast(loader, q, handler)
}

//OnboardRemove remove onboard by onboardUUID
func (adam *Ctx) OnboardRemove(onboardUUID string) (err error) {
	return adam.deleteObj(path.Join("/admin/onboard", onboardUUID))
}

//DeviceRemove remove device by devUUID
func (adam *Ctx) DeviceRemove(devUUID uuid.UUID) (err error) {
	return adam.deleteObj(path.Join("/admin/device", devUUID.String()))
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
	return adam.DeviceGetByOnboardUUID(uuidToFound.String())
}

//DeviceGetByOnboardUUID try to get device by onboard uuid
func (adam *Ctx) DeviceGetByOnboardUUID(onboardUUID string) (devUUID uuid.UUID, err error) {
	devIDs, err := adam.DeviceList(types.RegisteredDeviceFilter)
	if err != nil {
		return uuid.Nil, err
	}
	for _, devID := range devIDs {
		devUUID, err := uuid.FromString(devID)
		if err != nil {
			return uuid.Nil, err
		}
		if id, err := adam.DeviceGetOnboard(devUUID); err == nil {
			if id.String() == onboardUUID {
				return devUUID, nil
			}
		} else {
			return uuid.Nil, err
		}
	}
	return uuid.Nil, fmt.Errorf("no device found")
}
