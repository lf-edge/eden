package utils

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"text/template"

	"github.com/lf-edge/eden/pkg/defaults"
	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var viperAccessMutex sync.RWMutex

//ConfigVars struct with parameters from config file
type ConfigVars struct {
	AdamIP            string
	AdamPort          string
	AdamDomain        string
	AdamDir           string
	AdamCA            string
	AdamRemote        bool
	AdamCaching       bool
	AdamCachingRedis  bool
	AdamCachingPrefix string
	AdamRemoteRedis   bool
	AdamRedisURLEden  string
	AdamRedisURLAdam  string
	EveHV             string
	EveSSID           string
	EveUUID           string
	EveName           string
	EveRemote         bool
	EveRemoteAddr     string
	EveQemuPorts      map[string]string
	EveQemuConfig     string
	EveDist           string
	SSHKey            string
	EveCert           string
	EveDeviceCert     string
	EveSerial         string
	ZArch             string
	DevModel          string
	DevModelFIle      string
	EdenBinDir        string
	EdenProg          string
	TestProg          string
	TestScenario      string
	EServerImageDist  string
	EServerPort       string
	EServerIP         string
	RegistryIP        string
	RegistryPort      string
	LogLevel          string
	AdamLogLevel      string
}

//InitVars loads vars from viper
func InitVars() (*ConfigVars, error) {
	loaded := true
	if viper.ConfigFileUsed() == "" {
		configPath, err := DefaultConfigPath()
		if err != nil {
			return nil, err
		}
		loaded, err = LoadConfigFile(configPath)
		if err != nil {
			return nil, err
		}
	}
	if loaded {
		edenHome, err := DefaultEdenDir()
		if err != nil {
			log.Fatal(err)
		}
		globalCertsDir := filepath.Join(edenHome, defaults.DefaultCertsDist)
		if _, err := os.Stat(globalCertsDir); os.IsNotExist(err) {
			if err = os.MkdirAll(globalCertsDir, 0755); err != nil {
				log.Fatal(err)
			}
		}
		caCertPath := filepath.Join(globalCertsDir, "root-certificate.pem")
		viperAccessMutex.RLock()
		var vars = &ConfigVars{
			AdamIP:            viper.GetString("adam.ip"),
			AdamPort:          viper.GetString("adam.port"),
			AdamDomain:        viper.GetString("adam.domain"),
			AdamDir:           ResolveAbsPath(viper.GetString("adam.dist")),
			AdamCA:            caCertPath,
			AdamRedisURLEden:  viper.GetString("adam.redis.eden"),
			SSHKey:            ResolveAbsPath(viper.GetString("eden.ssh-key")),
			EveCert:           ResolveAbsPath(viper.GetString("eve.cert")),
			EveDeviceCert:     ResolveAbsPath(viper.GetString("eve.device-cert")),
			EveSerial:         viper.GetString("eve.serial"),
			EveDist:           viper.GetString("eve.dist"),
			EveQemuConfig:     viper.GetString("eve.qemu-config"),
			ZArch:             viper.GetString("eve.arch"),
			EveSSID:           viper.GetString("eve.ssid"),
			EveHV:             viper.GetString("eve.hv"),
			DevModel:          viper.GetString("eve.devmodel"),
			DevModelFIle:      viper.GetString("eve.devmodelfile"),
			EveName:           viper.GetString("eve.name"),
			EveUUID:           viper.GetString("eve.uuid"),
			EveRemote:         viper.GetBool("eve.remote"),
			EveRemoteAddr:     viper.GetString("eve.remote-addr"),
			EveQemuPorts:      viper.GetStringMapString("eve.hostfwd"),
			AdamRemote:        viper.GetBool("adam.remote.enabled"),
			AdamRemoteRedis:   viper.GetBool("adam.remote.redis"),
			AdamCaching:       viper.GetBool("adam.caching.enabled"),
			AdamCachingPrefix: viper.GetString("adam.caching.prefix"),
			AdamCachingRedis:  viper.GetBool("adam.caching.redis"),
			EdenBinDir:        viper.GetString("eden.bin-dist"),
			EdenProg:          viper.GetString("eden.eden-bin"),
			TestProg:          viper.GetString("eden.test-bin"),
			TestScenario:      viper.GetString("eden.test-scenario"),
			EServerImageDist:  ResolveAbsPath(viper.GetString("eden.images.dist")),
			EServerPort:       viper.GetString("eden.eserver.port"),
			EServerIP:         viper.GetString("eden.eserver.ip"),
			RegistryIP:        viper.GetString("registry.ip"),
			RegistryPort:      viper.GetString("registry.port"),
			LogLevel:          viper.GetString("eve.log-level"),
			AdamLogLevel:      viper.GetString("eve.adam-log-level"),
		}
		viperAccessMutex.RUnlock()
		redisPasswordFile := filepath.Join(globalCertsDir, defaults.DefaultRedisPasswordFile)
		pwd, err := ioutil.ReadFile(redisPasswordFile)
		if err == nil {
			vars.AdamRedisURLEden = fmt.Sprintf("redis://%s:%s@%s", string(pwd), string(pwd), vars.AdamRedisURLEden)
		} else {
			log.Errorf("cannot read redis password: %v", err)
			vars.AdamRedisURLEden = fmt.Sprintf("redis://%s", vars.AdamRedisURLEden)
		}
		return vars, nil
	}
	return nil, nil
}

//DefaultEdenDir returns path to default directory
func DefaultEdenDir() (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", err
	}
	return filepath.Join(usr.HomeDir, defaults.DefaultEdenHomeDir), nil
}

//GetConfig return path to config file
func GetConfig(name string) string {
	edenDir, err := DefaultEdenDir()
	if err != nil {
		log.Fatalf("GetCurrentConfig DefaultEdenDir error: %s", err)
	}
	return filepath.Join(edenDir, defaults.DefaultContextDirectory, fmt.Sprintf("%s.yml", name))
}

//DefaultConfigPath returns path to default config
func DefaultConfigPath() (string, error) {
	context, err := ContextLoad()
	if err != nil {
		return "", fmt.Errorf("context load error: %s", err)
	}
	return context.GetCurrentConfig(), nil
}

//CurrentDirConfigPath returns path to eden-config.yml in current folder
func CurrentDirConfigPath() (string, error) {
	currentPath, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return filepath.Join(currentPath, defaults.DefaultCurrentDirConfig), nil
}

func loadConfigFile(config string, local bool) (loaded bool, err error) {
	if config == "" {
		config, err = DefaultConfigPath()
		if err != nil {
			return false, fmt.Errorf("fail in DefaultConfigPath: %s", err.Error())
		}
	} else {
		context, err := ContextInit()
		if err != nil {
			return false, fmt.Errorf("context Load DefaultEdenDir error: %s", err)
		}
		contextFile := context.GetCurrentConfig()
		if config != contextFile {
			loaded, err := loadConfigFile(contextFile, true)
			if err != nil {
				return loaded, err
			}
		}
	}
	log.Debugf("Will use config from %s", config)
	if _, err = os.Stat(config); os.IsNotExist(err) {
		log.Fatal("no config, please run 'eden config add'")
	}
	abs, err := filepath.Abs(config)
	if err != nil {
		return false, fmt.Errorf("fail in reading filepath: %s", err.Error())
	}
	viper.SetConfigFile(abs)
	if err := viper.MergeInConfig(); err != nil {
		return false, fmt.Errorf("failed to read config file: %s", err.Error())
	}
	if local {
		currentFolderDir, err := CurrentDirConfigPath()
		if err != nil {
			log.Errorf("CurrentDirConfigPath: %s", err)
		} else {
			log.Debugf("Try to add config from %s", currentFolderDir)
			if _, err = os.Stat(currentFolderDir); !os.IsNotExist(err) {
				abs, err = filepath.Abs(currentFolderDir)
				if err != nil {
					log.Errorf("CurrentDirConfigPath absolute: %s", err)
				} else {
					viper.SetConfigFile(abs)
					if err := viper.MergeInConfig(); err != nil {
						log.Errorf("failed in merge config file: %s", err.Error())
					} else {
						log.Debugf("Merged config with %s", abs)
					}
				}
			}
		}
	}
	return true, nil
}

//LoadConfigFile load config from file with viper
func LoadConfigFile(config string) (loaded bool, err error) {
	viperAccessMutex.Lock()
	defer viperAccessMutex.Unlock()
	return loadConfigFile(config, true)
}

//LoadConfigFileContext load config from context file with viper
func LoadConfigFileContext(config string) (loaded bool, err error) {
	viperAccessMutex.Lock()
	defer viperAccessMutex.Unlock()
	return loadConfigFile(config, false)
}

//GenerateConfigFile is a function to generate default yml
func GenerateConfigFile(filePath string) error {
	context, err := ContextInit()
	if err != nil {
		return err
	}
	context.Save()
	return generateConfigFileFromTemplate(filePath, defaults.DefaultEdenTemplate, context)
}

func generateConfigFileFromTemplate(filePath string, templateString string, context *Context) error {
	currentPath, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		log.Fatal(err)
	}
	file, err := os.Create(filePath)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	edenDir, err := DefaultEdenDir()
	if err != nil {
		log.Fatal(err)
	}

	ip, err := GetIPForDockerAccess()
	if err != nil {
		return err
	}
	id, err := uuid.NewV4()
	if err != nil {
		return err
	}

	imageDist := fmt.Sprintf("%s-%s", context.Current, defaults.DefaultImageDist)

	certsDist := fmt.Sprintf("%s-%s", context.Current, defaults.DefaultCertsDist)

	parse := func(inp string) interface{} {
		switch inp {
		case "adam.tag":
			return defaults.DefaultAdamTag
		case "adam.dist":
			return defaults.DefaultAdamDist
		case "adam.port":
			return defaults.DefaultAdamPort
		case "adam.domain":
			return defaults.DefaultDomain
		case "adam.eve-ip":
			return ip
		case "adam.ip":
			return ip
		case "adam.redis.eden":
			return fmt.Sprintf("%s:%d", ip, defaults.DefaultRedisPort)
		case "adam.redis.adam":
			return fmt.Sprintf("%s:%d", defaults.DefaultRedisContainerName, defaults.DefaultRedisPort)
		case "adam.force":
			return true
		case "adam.ca":
			return filepath.Join(certsDist, "root-certificate.pem")
		case "adam.remote.enabled":
			return true
		case "adam.remote.redis":
			return true
		case "adam.v1":
			return false
		case "adam.caching.enabled":
			return false
		case "adam.caching.redis":
			return false
		case "adam.caching.prefix":
			return "cache"

		case "eve.name":
			return strings.ToLower(context.Current)
		case "eve.devmodel":
			return defaults.DefaultQemuModel
		case "eve.devmodelfile":
			return ""
		case "eve.arch":
			return runtime.GOARCH
		case "eve.os":
			return runtime.GOOS
		case "eve.accel":
			return true
		case "eve.hv":
			return defaults.DefaultEVEHV
		case "eve.serial":
			return defaults.DefaultEVESerial
		case "eve.cert":
			return filepath.Join(certsDist, "onboard.cert.pem")
		case "eve.device-cert":
			return filepath.Join(certsDist, "device.cert.pem")
		case "eve.pid":
			return fmt.Sprintf("%s-eve.pid", strings.ToLower(context.Current))
		case "eve.log":
			return fmt.Sprintf("%s-eve.log", strings.ToLower(context.Current))
		case "eve.firmware":
			if runtime.GOARCH == "amd64" {
				return fmt.Sprintf("[%s %s]",
					filepath.Join(imageDist, "eve", "OVMF_CODE.fd"),
					filepath.Join(imageDist, "eve", "OVMF_VARS.fd"))
			}
			return fmt.Sprintf("[%s]", filepath.Join(imageDist, "eve", "OVMF.fd"))
		case "eve.repo":
			return defaults.DefaultEveRepo
		case "eve.registry":
			return defaults.DefaultEveRegistry
		case "eve.tag":
			return defaults.DefaultEVETag
		case "eve.hostfwd":
			return fmt.Sprintf("{\"%d\":\"22\",\"5912\":\"5902\",\"5911\":\"5901\",\"8027\":\"8027\",\"8028\":\"8028\"}", defaults.DefaultSSHPort)
		case "eve.dist":
			return fmt.Sprintf("%s-%s", context.Current, defaults.DefaultEVEDist)
		case "eve.qemu-config":
			return filepath.Join(edenDir, fmt.Sprintf("%s-%s", context.Current, defaults.DefaultQemuFileToSave))
		case "eve.uuid":
			return id.String()
		case "eve.image-file":
			return filepath.Join(imageDist, "eve", "live.img")
		case "eve.dtb-part":
			return ""
		case "eve.config-part":
			return certsDist
		case "eve.remote":
			return defaults.DefaultEVERemote
		case "eve.remote-addr":
			return defaults.DefaultEVEHost
		case "eve.log-level":
			return defaults.DefaultEveLogLevel
		case "eve.adam-log-level":
			return defaults.DefaultAdamLogLevel
		case "eve.telnet-port":
			return defaults.DefaultTelnetPort
		case "eve.ssid":
			return ""
		case "eve.qemu.monitor-port":
			return defaults.DefaultQemuMonitorPort
		case "eve.qemu.netdev-socket-port":
			return defaults.DefaultQemuNetdevSocketPort
		case "eve.cpu":
			return defaults.DefaultCpus
		case "eve.ram":
			return defaults.DefaultMemory
		case "eve.disk":
			return defaults.DefaultEVEImageSize
		case "eve.tpm":
			return defaults.DefaultTPMEnabled
		case "eve.disks":
			return defaults.DefaultAdditionalDisks

		case "eden.root":
			return filepath.Join(currentPath, defaults.DefaultDist)
		case "eden.tests":
			return filepath.Join(currentPath, defaults.DefaultDist, "tests")
		case "eden.images.dist":
			return defaults.DefaultEserverDist
		case "eden.download":
			return true
		case "eden.eserver.eve-ip":
			return defaults.DefaultDomain
		case "eden.eserver.ip":
			return ip
		case "eden.eserver.port":
			return defaults.DefaultEserverPort
		case "eden.eserver.tag":
			return defaults.DefaultEServerTag
		case "eden.eserver.force":
			return true
		case "eden.eclient.tag":
			return defaults.DefaultEClientTag
		case "eden.eclient.image":
			return defaults.DefaultEClientContainerRef
		case "eden.certs-dist":
			return certsDist
		case "eden.bin-dist":
			return defaults.DefaultBinDist
		case "eden.ssh-key":
			return fmt.Sprintf("%s-%s", context.Current, defaults.DefaultSSHKey)
		case "eden.eden-bin":
			return "eden"
		case "eden.test-bin":
			return defaults.DefaultTestProg
		case "eden.test-scenario":
			return defaults.DefaultTestScenario

		case "gcp.key":
			return ""

		case "packet.key":
			return ""

		case "redis.port":
			return defaults.DefaultRedisPort
		case "redis.tag":
			return defaults.DefaultRedisTag
		case "redis.dist":
			return defaults.DefaultRedisDist

		case "registry.port":
			return defaults.DefaultRegistryPort
		case "registry.tag":
			return defaults.DefaultRegistryTag
		case "registry.ip":
			return ip
		case "registry.dist":
			return defaults.DefaultRegistryDist

		case "sdn.image-file":
			return filepath.Join(imageDist, "eden", "sdn-efi.qcow2")
		case "sdn.pid":
			return filepath.Join(currentPath, defaults.DefaultDist, "sdn.pid")
		case "sdn.console-log":
			return filepath.Join(currentPath, defaults.DefaultDist, "sdn-console.log")
		case "sdn.telnet-port":
			return defaults.DefaultSdnTelnetPort
		case "sdn.ssh-port":
			return defaults.DefaultSdnSSHPort
		case "sdn.mgmt-port":
			return defaults.DefaultSdnMgmtPort
		case "sdn.network-model":
			return ""

		default:
			log.Fatalf("Not found argument %s in config", inp)
		}
		return ""
	}
	var fm = template.FuncMap{
		"parse": parse,
	}
	t := template.New("t").Funcs(fm)
	_, err = t.Parse(templateString)
	if err != nil {
		return err
	}
	buf := new(bytes.Buffer)
	err = t.Execute(buf, nil)
	if err != nil {
		return err
	}
	_, err = file.Write(buf.Bytes())
	if err != nil {
		log.Fatal(err)
	}
	return nil
}

func generateConfigFileFromViperTemplate(filePath string, templateString string) error {
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		log.Fatal(err)
	}
	file, err := os.Create(filePath)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	parse := func(inp string) interface{} {
		result := viper.Get(inp)
		if result != nil {
			return result
		}
		log.Warnf("Not found argument %s in config", inp)
		return ""
	}
	var fm = template.FuncMap{
		"parse": parse,
	}
	t := template.New("t").Funcs(fm)
	_, err = t.Parse(templateString)
	if err != nil {
		return err
	}
	buf := new(bytes.Buffer)
	err = t.Execute(buf, nil)
	if err != nil {
		return err
	}
	_, err = file.Write(buf.Bytes())
	if err != nil {
		log.Fatal(err)
	}
	return nil
}

//GenerateConfigFileFromViper is a function to generate yml from viper config
func GenerateConfigFileFromViper() error {
	configFile, err := DefaultConfigPath()
	if err != nil {
		log.Fatalf("fail in DefaultConfigPath: %s", err)
	}
	return generateConfigFileFromViperTemplate(configFile, defaults.DefaultEdenTemplate)
}

//GenerateConfigFileDiff is a function to generate diff yml for new context
func GenerateConfigFileDiff(filePath string, context *Context) error {
	return generateConfigFileFromTemplate(filePath, defaults.DefaultEdenTemplate, context)
}
