package adam

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/lf-edge/eden/pkg/controller/cachers"
	"github.com/lf-edge/eden/pkg/controller/eapps"
	"github.com/lf-edge/eden/pkg/controller/eflowlog"
	"github.com/lf-edge/eden/pkg/controller/einfo"
	"github.com/lf-edge/eden/pkg/controller/elog"
	"github.com/lf-edge/eden/pkg/controller/emetric"
	"github.com/lf-edge/eden/pkg/controller/erequest"
	"github.com/lf-edge/eden/pkg/controller/loaders"
	"github.com/lf-edge/eden/pkg/controller/types"
	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/device"
	"github.com/lf-edge/eden/pkg/utils"
	"github.com/lf-edge/eve-api/go/auth"
	"github.com/lf-edge/eve-api/go/certs"
	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"
)

const (
	mimeProto     = "application/x-proto-binary"
	mimeTextPlain = "text/plain"
	mimeJSON      = "application/json"
)

// Ctx stores controller settings
type Ctx struct {
	dir               string
	url               string
	serverCA          string
	insecureTLS       bool
	AdamRemote        bool
	AdamRemoteRedis   bool   // use redis for obtain logs and info
	AdamRedisURLEden  string // string with redis url for obtain logs and info
	AdamCaching       bool   // enable caching of adam`s logs/info
	AdamCachingRedis  bool   // caching to redis instead of files
	AdamCachingPrefix string // custom prefix for file or stream naming for cache
}

// parseRedisURL try to use string from config to obtain redis url
func parseRedisURL(s string) (addr, password string, databaseID int, err error) {
	URL, err := url.Parse(s)
	if err != nil || URL.Scheme != "redis" {
		return "", "", 0, err
	}

	if URL.Host != "" {
		addr = URL.Host
	} else {
		addr = fmt.Sprintf("%s:%d", defaults.DefaultRedisHost, defaults.DefaultRedisPort)
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

// getLoader return loader object from Adam`s config
func (adam *Ctx) getLoader() (loader loaders.Loader) {
	if adam.AdamRemote {
		log.Debug("will use remote adam loader")
		if adam.AdamRemoteRedis {
			addr, password, databaseID, err := parseRedisURL(adam.AdamRedisURLEden)
			if err != nil {
				log.Fatalf("Cannot parse adam redis url: %s", err)
			}
			streamGetters := types.StreamGetters{
				StreamLogs:    adam.getLogsRedisStream,
				StreamInfo:    adam.getInfoRedisStream,
				StreamMetrics: adam.getMetricsRedisStream,
				StreamFlowLog: adam.getFlowLogRedisStream,
				StreamRequest: adam.getRequestRedisStream,
				StreamApps:    adam.getAppsLogsRedisStream,
			}
			loader = loaders.NewRedisLoader(addr, password, databaseID, streamGetters)
		} else {
			urlGetters := types.URLGetters{
				URLLogs:    adam.getLogsURL,
				URLInfo:    adam.getInfoURL,
				URLMetrics: adam.getMetricsURL,
				URLRequest: adam.getRequestURL,
			}
			loader = loaders.NewRemoteLoader(adam.getHTTPClient, urlGetters)
		}
	} else {
		log.Debug("will use local adam loader")
		dirGetters := types.DirGetters{
			LogsGetter:    adam.getLogsDir,
			InfoGetter:    adam.getInfoDir,
			MetricsGetter: adam.getMetricsDir,
			RequestGetter: adam.getRequestDir,
		}
		loader = loaders.NewFileLoader(dirGetters)
	}
	if adam.AdamCaching {
		var cache cachers.CacheProcessor
		if adam.AdamCachingRedis {
			addr, password, databaseID, err := parseRedisURL(adam.AdamRedisURLEden)
			if err != nil {
				log.Fatalf("Cannot parse adam redis url: %s", err)
			}
			streamGetters := types.StreamGetters{
				StreamLogs:    adam.getLogsRedisStreamCache,
				StreamInfo:    adam.getInfoRedisStreamCache,
				StreamMetrics: adam.getMetricsRedisStreamCache,
				StreamRequest: adam.getRequestRedisStreamCache,
			}
			cache = cachers.NewRedisCache(addr, password, databaseID, streamGetters)
		} else {
			dirGetters := types.DirGetters{
				LogsGetter:    adam.getLogsDirCache,
				InfoGetter:    adam.getInfoDirCache,
				MetricsGetter: adam.getMetricsDirCache,
				RequestGetter: adam.getRequestDirCache,
			}
			cache = cachers.NewFileCache(dirGetters)
		}
		loader.SetRemoteCache(cache)
	}
	return
}

// InitWithVars use variables from viper for init controller
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
	adam.AdamRedisURLEden = vars.AdamRedisURLEden
	return nil
}

// GetDir return dir
func (adam *Ctx) GetDir() (dir string) {
	return adam.dir
}

// Register device in adam
func (adam *Ctx) Register(device *device.Ctx) error {
	b, err := os.ReadFile(device.GetOnboardKey())
	switch {
	case err != nil && os.IsNotExist(err):
		log.Printf("cert file %s does not exist", device.GetOnboardKey())
		return err
	case err != nil:
		log.Printf("error reading cert file %s: %v", device.GetOnboardKey(), err)
		return err
	}

	objToSend := types.OnboardCert{
		Cert:   b,
		Serial: device.GetSerial(),
	}
	body, err := json.Marshal(objToSend)
	if err != nil {
		log.Printf("error encoding json: %v", err)
		return err
	}
	return adam.postObj("/admin/onboard", body, mimeJSON)
}

// DeviceList return device list
func (adam *Ctx) DeviceList(filter types.DeviceStateFilter) (out []string, err error) {
	if filter == types.RegisteredDeviceFilter || filter == types.AllDevicesFilter {
		return adam.getList("/admin/device", mimeJSON)
	}
	return []string{}, nil
}

// ConfigSet set config for devID
func (adam *Ctx) ConfigSet(devUUID uuid.UUID, devConfig []byte) (err error) {
	return adam.putObj(path.Join("/admin/device", devUUID.String(), "config"), devConfig, mimeProto)
}

// ConfigGet get config for devID
func (adam *Ctx) ConfigGet(devUUID uuid.UUID) (out string, err error) {
	return adam.getObj(path.Join("/admin/device", devUUID.String(), "config"), mimeProto)
}

// GetECDHCert get cert for ECDH exchange for devID
func (adam *Ctx) GetECDHCert(devUUID uuid.UUID) ([]byte, error) {
	attestData, err := adam.getObj(path.Join("/admin/device", devUUID.String(), "certs"), mimeJSON)
	if err != nil {
		return nil, fmt.Errorf("cannot get attestation certificates from cloud for %s", devUUID)
	}
	req := &types.Zcerts{}
	if err := json.Unmarshal([]byte(attestData), req); err != nil {
		return nil, fmt.Errorf("cannot unmarshal attest: %w", err)
	}
	var devCert []byte
	for _, c := range req.Certs {
		if c.Type == certs.ZCertType_CERT_TYPE_DEVICE_ECDH_EXCHANGE {
			devCert = c.Cert
		}
	}
	if len(devCert) == 0 {
		return nil, fmt.Errorf("no DEVICE_ECDH_EXCHANGE certificate")
	}
	return devCert, nil
}

// RequestLastCallback check request by pattern from existence files with callback
func (adam *Ctx) RequestLastCallback(devUUID uuid.UUID, q map[string]string, handler erequest.HandlerFunc) (err error) {
	var loader = adam.getLoader()
	loader.SetUUID(devUUID)
	return erequest.RequestLast(loader, q, handler)
}

// LogAppsChecker check app logs by pattern from existence files with LogLast and use LogWatchWithTimeout with timeout for observe new files
func (adam *Ctx) LogAppsChecker(devUUID uuid.UUID, appUUID uuid.UUID, q map[string]string, handler eapps.HandlerFunc, mode eapps.LogCheckerMode, timeout time.Duration) (err error) {
	return eapps.LogChecker(adam.getLoader(), devUUID, appUUID, q, handler, mode, timeout)
}

// LogAppsLastCallback check app logs by pattern from existence files with callback
func (adam *Ctx) LogAppsLastCallback(devUUID uuid.UUID, appUUID uuid.UUID, q map[string]string, handler eapps.HandlerFunc) (err error) {
	var loader = adam.getLoader()
	loader.SetUUID(devUUID)
	loader.SetAppUUID(appUUID)
	return eapps.LogLast(loader, q, handler)
}

// LogChecker check logs by pattern from existence files with LogLast and use LogWatchWithTimeout with timeout for observe new files
func (adam *Ctx) LogChecker(devUUID uuid.UUID, q map[string]string, handler elog.HandlerFunc, mode elog.LogCheckerMode, timeout time.Duration) (err error) {
	return elog.LogChecker(adam.getLoader(), devUUID, q, handler, mode, timeout)
}

// LogLastCallback check logs by pattern from existence files with callback
func (adam *Ctx) LogLastCallback(devUUID uuid.UUID, q map[string]string, handler elog.HandlerFunc) (err error) {
	var loader = adam.getLoader()
	loader.SetUUID(devUUID)
	return elog.LogLast(loader, q, handler)
}

// FlowLogChecker check FlowLogs by pattern from existence files with FlowLogLast and use FlowLogWatchWithTimeout with timeout for observe new files
func (adam *Ctx) FlowLogChecker(devUUID uuid.UUID, q map[string]string, handler eflowlog.HandlerFunc, mode eflowlog.FlowLogCheckerMode, timeout time.Duration) (err error) {
	return eflowlog.FlowLogChecker(adam.getLoader(), devUUID, q, handler, mode, timeout)
}

// FlowLogLastCallback check FlowLogs by pattern from existence files with callback
func (adam *Ctx) FlowLogLastCallback(devUUID uuid.UUID, q map[string]string, handler eflowlog.HandlerFunc) (err error) {
	var loader = adam.getLoader()
	loader.SetUUID(devUUID)
	return eflowlog.FlowLogLast(loader, q, handler)
}

// InfoChecker checks the information in the regular expression pattern 'query' and processes the info.ZInfoMsg found by the function 'handler' from existing files (mode=einfo.InfoExist), new files (mode=einfo.InfoNew) or any of them (mode=einfo.InfoAny) with timeout.
func (adam *Ctx) InfoChecker(devUUID uuid.UUID, q map[string]string, handler einfo.HandlerFunc, mode einfo.InfoCheckerMode, timeout time.Duration) (err error) {
	return einfo.InfoChecker(adam.getLoader(), devUUID, q, handler, mode, timeout)
}

// InfoLastCallback check info by pattern from existence files with callback
func (adam *Ctx) InfoLastCallback(devUUID uuid.UUID, q map[string]string, handler einfo.HandlerFunc) (err error) {
	var loader = adam.getLoader()
	loader.SetUUID(devUUID)
	return einfo.InfoLast(loader, q, einfo.ZInfoFind, handler)
}

// MetricChecker check metrics by pattern from existence files with LogLast and use LogWatchWithTimeout with timeout for observe new files
func (adam *Ctx) MetricChecker(devUUID uuid.UUID, q map[string]string, handler emetric.HandlerFunc, mode emetric.MetricCheckerMode, timeout time.Duration) (err error) {
	return emetric.MetricChecker(adam.getLoader(), devUUID, q, handler, mode, timeout)
}

// MetricLastCallback check metrics by pattern from existence files with callback
func (adam *Ctx) MetricLastCallback(devUUID uuid.UUID, q map[string]string, handler emetric.HandlerFunc) (err error) {
	var loader = adam.getLoader()
	loader.SetUUID(devUUID)
	return emetric.MetricLast(loader, q, handler)
}

// OnboardRemove remove onboard by onboardUUID
func (adam *Ctx) OnboardRemove(onboardUUID string) (err error) {
	return adam.deleteObj(path.Join("/admin/onboard", onboardUUID))
}

// DeviceRemove remove device by devUUID
func (adam *Ctx) DeviceRemove(devUUID uuid.UUID) (err error) {
	return adam.deleteObj(path.Join("/admin/device", devUUID.String()))
}

// DeviceGetOnboard get device onboardUUID for devUUID
func (adam *Ctx) DeviceGetOnboard(devUUID uuid.UUID) (onboardUUID uuid.UUID, err error) {
	var devCert types.DeviceCert
	devInfo, err := adam.getObj(path.Join("/admin/device", devUUID.String()), mimeJSON)
	if err != nil {
		return uuid.Nil, err
	}
	if err = json.Unmarshal([]byte(devInfo), &devCert); err != nil {
		return uuid.Nil, err
	}
	cert, err := utils.ParseFirstCertFromBlock(devCert.Onboard)
	if err != nil {
		return uuid.Nil, err
	}
	return uuid.FromString(cert.Subject.CommonName)
}

// DeviceGetByOnboard try to get device by onboard eveCert
func (adam *Ctx) DeviceGetByOnboard(eveCert string) (devUUID uuid.UUID, err error) {
	b, err := os.ReadFile(eveCert)
	switch {
	case err != nil && os.IsNotExist(err):
		log.Printf("cert file %s does not exist", eveCert)
		return uuid.Nil, err
	case err != nil:
		log.Printf("error reading cert file %s: %v", eveCert, err)
		return uuid.Nil, err
	}
	cert, err := utils.ParseFirstCertFromBlock(b)
	if err != nil {
		return uuid.Nil, err
	}
	uuidToFound, err := uuid.FromString(cert.Subject.CommonName)
	if err != nil {
		return uuid.Nil, err
	}
	return adam.DeviceGetByOnboardUUID(uuidToFound.String())
}

// DeviceGetByOnboardUUID try to get device by onboard uuid
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

// GetDeviceCert gets deviceCert contains certificates and serial
func (adam *Ctx) GetDeviceCert(device *device.Ctx) (deviceCert *types.DeviceCert, err error) {
	devInfo, err := adam.getObj(path.Join("/admin/device", device.GetID().String()), mimeJSON)
	if err != nil {
		return nil, err
	}
	var devCert types.DeviceCert
	if err = json.Unmarshal([]byte(devInfo), &devCert); err != nil {
		return nil, err
	}
	return &devCert, nil
}

// UploadDeviceCert upload deviceCert into Adam
func (adam *Ctx) UploadDeviceCert(deviceCert types.DeviceCert) error {
	body, err := json.Marshal(deviceCert)
	if err != nil {
		return err
	}
	return adam.postObj("/admin/device", body, mimeTextPlain)
}

// SetDeviceOptions sets options for provided devUUID
func (adam *Ctx) SetDeviceOptions(devUUID uuid.UUID, options *types.DeviceOptions) error {
	body, err := json.Marshal(options)
	if err != nil {
		return err
	}
	return adam.putObj(path.Join("/admin/device", devUUID.String(), "options"), body, mimeJSON)
}

// GetDeviceOptions returns DeviceOptions for provided devUUID
func (adam *Ctx) GetDeviceOptions(devUUID uuid.UUID) (*types.DeviceOptions, error) {
	devInfo, err := adam.getObj(path.Join("/admin/device", devUUID.String(), "options"), mimeJSON)
	if err != nil {
		return nil, err
	}
	var devOptions types.DeviceOptions
	if err = json.Unmarshal([]byte(devInfo), &devOptions); err != nil {
		return nil, err
	}
	return &devOptions, nil
}

// SetGlobalOptions sets global options for controller
func (adam *Ctx) SetGlobalOptions(options *types.GlobalOptions) error {
	body, err := json.Marshal(options)
	if err != nil {
		return err
	}
	return adam.putObj("/admin/options", body, mimeJSON)
}

// GetGlobalOptions returns global options from controller
func (adam *Ctx) GetGlobalOptions() (*types.GlobalOptions, error) {
	devInfo, err := adam.getObj("/admin/options", mimeJSON)
	if err != nil {
		return nil, err
	}
	var globalOptions types.GlobalOptions
	if err = json.Unmarshal([]byte(devInfo), &globalOptions); err != nil {
		return nil, err
	}
	return &globalOptions, nil
}

// SigningCertGet gets signing certificate from Adam
func (adam *Ctx) SigningCertGet() (signCert []byte, err error) {
	certsData, err := adam.getObj("/api/v2/edgedevice/certs", mimeProto)
	if err != nil {
		return nil, err
	}
	zcloudMsg := &auth.AuthContainer{}
	err = proto.Unmarshal([]byte(certsData), zcloudMsg)
	if err != nil {
		return nil, err
	}
	ctrlCert := &certs.ZControllerCert{}
	err = proto.Unmarshal(zcloudMsg.ProtectedPayload.Payload, ctrlCert)
	if err != nil {
		return nil, err
	}
	for _, c := range ctrlCert.Certs {
		// there should be only one signing certificate, so we return the first one we find
		if c.Type == certs.ZCertType_CERT_TYPE_CONTROLLER_SIGNING {
			return c.Cert, nil
		}
	}
	return nil, fmt.Errorf("no signing certificate found")
}
