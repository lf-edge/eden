package openevec

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"

	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/tests"
	"github.com/lf-edge/eden/pkg/utils"
	log "github.com/sirupsen/logrus"
)

type TestArgs struct {
	TestArgs     string
	TestOpts     bool
	TestEscript  string
	TestRun      string
	TestTimeout  string
	TestList     string
	TestProg     string
	TestScenario string
	FailScenario string
	CurDir       string
	ConfigFile   string
	Verbosity    string
}

func InitVarsFromConfig(cfg *EdenSetupArgs) (*utils.ConfigVars, error) {
	var cv utils.ConfigVars
	edenHome, err := utils.DefaultEdenDir()
	if err != nil {
		return nil, err
	}
	globalCertsDir := filepath.Join(edenHome, defaults.DefaultCertsDist)
	if _, err := os.Stat(globalCertsDir); os.IsNotExist(err) {
		if err = os.MkdirAll(globalCertsDir, 0755); err != nil {
			return nil, err
		}
	}
	caCertPath := filepath.Join(globalCertsDir, "root-certificate.pem")

	cv.AdamIP = cfg.Adam.CertsIP
	cv.AdamPort = strconv.Itoa(cfg.Adam.Port)
	cv.AdamDomain = cfg.Adam.CertsDomain
	cv.AdamDir = cfg.Adam.Dist
	cv.AdamCA = caCertPath
	cv.AdamRedisURLEden = cfg.Adam.Redis.RemoteURL
	cv.AdamRemote = cfg.Adam.Remote.Enabled
	cv.AdamRemoteRedis = cfg.Adam.Remote.Redis
	cv.AdamCaching = cfg.Adam.Caching.Enabled
	cv.AdamCachingPrefix = cfg.Adam.Caching.Prefix
	cv.AdamCachingRedis = cfg.Adam.Caching.Redis

	cv.SSHKey = cfg.Eden.SSHKey
	cv.EdenBinDir = cfg.Eden.BinDir
	cv.EdenProg = cfg.Eden.EdenBin
	cv.TestProg = cfg.Eden.TestBin
	cv.TestScenario = cfg.Eden.TestScenario
	cv.EServerImageDist = cfg.Eden.Images.EServerImageDist
	cv.EServerPort = strconv.Itoa(cfg.Eden.EServer.Port)
	cv.EServerIP = cfg.Eden.EServer.IP

	cv.EveCert = cfg.Eve.Cert
	cv.EveDeviceCert = cfg.Eve.DeviceCert
	cv.EveSerial = cfg.Eve.Serial
	cv.EveDist = cfg.Eve.Dist
	cv.EveQemuConfig = cfg.Eve.QemuFileToSave
	cv.ZArch = cfg.Eve.Arch
	cv.EveSSID = cfg.Eve.Ssid
	cv.EveHV = cfg.Eve.HV
	cv.DevModel = cfg.Eve.DevModel
	cv.DevModelFIle = cfg.Eve.DevModelFile
	cv.EveName = cfg.Eve.Name
	cv.EveUUID = cfg.Eve.CertsUUID
	cv.AdamLogLevel = cfg.Eve.AdamLogLevel
	cv.EveRemote = cfg.Eve.Remote
	cv.EveRemoteAddr = cfg.Eve.RemoteAddr
	cv.EveQemuPorts = cfg.Eve.HostFwd
	cv.LogLevel = cfg.Eve.LogLevel

	cv.RegistryIP = cfg.Registry.IP
	cv.RegistryPort = strconv.Itoa(cfg.Registry.Port)

	redisPasswordFile := filepath.Join(globalCertsDir, defaults.DefaultRedisPasswordFile)
	pwd, err := ioutil.ReadFile(redisPasswordFile)
	if err == nil {
		cv.AdamRedisURLEden = fmt.Sprintf("redis://%s:%s@%s", string(pwd), string(pwd), cv.AdamRedisURLEden)
	} else {
		log.Errorf("cannot read redis password: %s", err.Error())
		cv.AdamRedisURLEden = fmt.Sprintf("redis://%s", cv.AdamRedisURLEden)
	}
	return &cv, nil
}

func Test(tstCfg *TestArgs) error {

	switch {
	case tstCfg.TestList != "":
		tests.RunTest(tstCfg.TestProg, []string{"-test.list", tstCfg.TestList}, "", tstCfg.TestTimeout, tstCfg.FailScenario, tstCfg.ConfigFile, tstCfg.Verbosity)
	case tstCfg.TestOpts:
		tests.RunTest(tstCfg.TestProg, []string{"-h"}, "", tstCfg.TestTimeout, tstCfg.FailScenario, tstCfg.ConfigFile, tstCfg.Verbosity)
	case tstCfg.TestEscript != "":
		tests.RunTest("eden.escript.test", []string{"-test.run", "TestEdenScripts/" + tstCfg.TestEscript}, tstCfg.TestArgs, tstCfg.TestTimeout, tstCfg.FailScenario, tstCfg.ConfigFile, tstCfg.Verbosity)
	case tstCfg.TestRun != "":
		tests.RunTest(tstCfg.TestProg, []string{"-test.run", tstCfg.TestRun}, tstCfg.TestArgs, tstCfg.TestTimeout, tstCfg.FailScenario, tstCfg.ConfigFile, tstCfg.Verbosity)
	default:
		tests.RunScenario(tstCfg.TestScenario, tstCfg.TestArgs, tstCfg.TestTimeout, tstCfg.FailScenario, tstCfg.ConfigFile, tstCfg.Verbosity)
	}

	if tstCfg.CurDir != "" {
		err := os.Chdir(tstCfg.CurDir)
		if err != nil {
			return err
		}
	}
	return nil
}
