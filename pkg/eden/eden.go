package eden

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"io/ioutil"
	"math/big"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/lf-edge/eden/eserver/api"
	"github.com/lf-edge/eden/pkg/controller"
	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/edensdn"
	"github.com/lf-edge/eden/pkg/models"
	"github.com/lf-edge/eden/pkg/utils"
	"github.com/nerd2/gexto"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

//StartRedis function run redis in docker with mounted redisPath:/data
//if redisForce is set, it recreates container
func StartRedis(redisPort int, redisPath string, redisForce bool, redisTag string) (err error) {
	portMap := map[string]string{"6379": strconv.Itoa(redisPort)}
	volumeMap := map[string]string{"/data": redisPath}
	redisServerCommand := strings.Fields("redis-server --appendonly yes")
	edenHome, err := utils.DefaultEdenDir()
	if err != nil {
		return err
	}
	globalCertsDir := filepath.Join(edenHome, defaults.DefaultCertsDist)
	redisPasswordFile := filepath.Join(globalCertsDir, defaults.DefaultRedisPasswordFile)
	pwd, err := ioutil.ReadFile(redisPasswordFile)
	if err == nil {
		redisServerCommand = append(redisServerCommand, strings.Fields(fmt.Sprintf("--requirepass %s", string(pwd)))...)
	} else {
		log.Errorf("cannot read redis password: %v", err)
	}
	if redisPath != "" {
		if err = os.MkdirAll(redisPath, 0755); err != nil {
			return fmt.Errorf("StartRedis: Cannot create directory for redis (%s): %s", redisPath, err)
		}
	}
	if redisForce {
		_ = utils.StopContainer(defaults.DefaultRedisContainerName, true)
		if err := utils.CreateAndRunContainer(defaults.DefaultRedisContainerName, defaults.DefaultRedisContainerRef+":"+redisTag, portMap, volumeMap, redisServerCommand, nil); err != nil {
			return fmt.Errorf("StartRedis: error in create redis container: %s", err)
		}
	} else {
		state, err := utils.StateContainer(defaults.DefaultRedisContainerName)
		if err != nil {
			return fmt.Errorf("StartRedis: error in get state of redis container: %s", err)
		}
		if state == "" {
			if err := utils.CreateAndRunContainer(defaults.DefaultRedisContainerName, defaults.DefaultRedisContainerRef+":"+redisTag, portMap, volumeMap, redisServerCommand, nil); err != nil {
				return fmt.Errorf("StartRedis: error in create redis container: %s", err)
			}
		} else if !strings.Contains(state, "running") {
			if err := utils.StartContainer(defaults.DefaultRedisContainerName); err != nil {
				return fmt.Errorf("StartRedis: error in restart redis container: %s", err)
			}
		}
	}
	return nil
}

//StopRedis function stop redis container
func StopRedis(redisRm bool) (err error) {
	state, err := utils.StateContainer(defaults.DefaultRedisContainerName)
	if err != nil {
		return fmt.Errorf("StopRedis: error in get state of redis container: %s", err)
	}
	if !strings.Contains(state, "running") {
		if redisRm {
			if err := utils.StopContainer(defaults.DefaultRedisContainerName, true); err != nil {
				return fmt.Errorf("StopRedis: error in rm redis container: %s", err)
			}
		}
	} else if state == "" {
		return nil
	} else {
		if redisRm {
			if err := utils.StopContainer(defaults.DefaultRedisContainerName, false); err != nil {
				return fmt.Errorf("StopRedis: error in rm redis container: %s", err)
			}
		} else {
			if err := utils.StopContainer(defaults.DefaultRedisContainerName, true); err != nil {
				return fmt.Errorf("StopRedis: error in rm redis container: %s", err)
			}
		}
	}
	return nil
}

//StatusRedis function return status of redis
func StatusRedis() (status string, err error) {
	state, err := utils.StateContainer(defaults.DefaultRedisContainerName)
	if err != nil {
		return "", fmt.Errorf("StatusRedis: error in get state of redis container: %s", err)
	}
	if state == "" {
		return "container doesn't exist", nil
	}
	return state, nil
}

//StartAdam function run adam in docker with mounted adamPath/run:/adam/run
//if adamForce is set, it recreates container
func StartAdam(adamPort int, adamPath string, adamForce bool, adamTag string, adamRemoteRedisURL string, apiV1 bool, opts ...string) (err error) {
	edenHome, err := utils.DefaultEdenDir()
	if err != nil {
		return err
	}
	globalCertsDir := filepath.Join(edenHome, defaults.DefaultCertsDist)
	serverCertPath := filepath.Join(globalCertsDir, "server.pem")
	serverKeyPath := filepath.Join(globalCertsDir, "server-key.pem")
	cert, err := ioutil.ReadFile(serverCertPath)
	if err != nil {
		return fmt.Errorf("StartAdam: cannot load %s: %s", serverCertPath, err)
	}
	key, err := ioutil.ReadFile(serverKeyPath)
	if err != nil {
		return fmt.Errorf("StartAdam: cannot load %s: %s", serverKeyPath, err)
	}
	envs := []string{
		fmt.Sprintf("SERVER_CERT=%s", cert),
		fmt.Sprintf("SERVER_KEY=%s", key),
	}
	if !apiV1 {
		signingCertPath := filepath.Join(globalCertsDir, "signing.pem")
		signingKeyPath := filepath.Join(globalCertsDir, "signing-key.pem")
		signingCert, err := ioutil.ReadFile(signingCertPath)
		if err != nil {
			return fmt.Errorf("StartAdam: cannot load %s: %s", signingCertPath, err)
		}
		signingKey, err := ioutil.ReadFile(signingKeyPath)
		if err != nil {
			return fmt.Errorf("StartAdam: cannot load %s: %s", signingKeyPath, err)
		}
		envs = append(envs, fmt.Sprintf("SIGNING_CERT=%s", signingCert))
		envs = append(envs, fmt.Sprintf("SIGNING_KEY=%s", signingKey))

		encryptCertPath := filepath.Join(globalCertsDir, "encrypt.pem")
		encryptKeyPath := filepath.Join(globalCertsDir, "encrypt-key.pem")
		encryptCert, err := ioutil.ReadFile(encryptCertPath)
		if err != nil {
			return fmt.Errorf("StartAdam: cannot load %s: %s", encryptCertPath, err)
		}
		encryptKey, err := ioutil.ReadFile(encryptKeyPath)
		if err != nil {
			return fmt.Errorf("StartAdam: cannot load %s: %s", encryptKeyPath, err)
		}
		envs = append(envs, fmt.Sprintf("ENCRYPT_CERT=%s", encryptCert))
		envs = append(envs, fmt.Sprintf("ENCRYPT_KEY=%s", encryptKey))
	}
	portMap := map[string]string{"8080": strconv.Itoa(adamPort)}
	volumeMap := map[string]string{"/adam/run": fmt.Sprintf("%s/run", adamPath)}
	adamServerCommand := strings.Fields("server --conf-dir ./run/conf")
	if adamPath == "" {
		volumeMap = map[string]string{"/adam/run": ""}
		adamServerCommand = strings.Fields("server")
	}
	if adamRemoteRedisURL != "" {
		redisPasswordFile := filepath.Join(globalCertsDir, defaults.DefaultRedisPasswordFile)
		pwd, err := ioutil.ReadFile(redisPasswordFile)
		if err == nil {
			adamRemoteRedisURL = fmt.Sprintf("redis://%s:%s@%s", string(pwd), string(pwd), adamRemoteRedisURL)
		} else {
			log.Errorf("cannot read redis password: %v", err)
			adamRemoteRedisURL = fmt.Sprintf("redis://%s", adamRemoteRedisURL)
		}
		adamServerCommand = append(adamServerCommand, strings.Fields(fmt.Sprintf("--db-url %s", adamRemoteRedisURL))...)
	}
	adamServerCommand = append(adamServerCommand, opts...)
	if adamForce {
		_ = utils.StopContainer(defaults.DefaultAdamContainerName, true)
		if err := utils.CreateAndRunContainer(defaults.DefaultAdamContainerName, defaults.DefaultAdamContainerRef+":"+adamTag, portMap, volumeMap, adamServerCommand, envs); err != nil {
			return fmt.Errorf("StartAdam: error in create adam container: %s", err)
		}
	} else {
		state, err := utils.StateContainer(defaults.DefaultAdamContainerName)
		if err != nil {
			return fmt.Errorf("StartAdam: error in get state of adam container: %s", err)
		}
		if state == "" {
			if err := utils.CreateAndRunContainer(defaults.DefaultAdamContainerName, defaults.DefaultAdamContainerRef+":"+adamTag, portMap, volumeMap, adamServerCommand, envs); err != nil {
				return fmt.Errorf("StartAdam: error in create adam container: %s", err)
			}
		} else if !strings.Contains(state, "running") {
			if err := utils.StartContainer(defaults.DefaultAdamContainerName); err != nil {
				return fmt.Errorf("StartAdam: error in restart adam container: %s", err)
			}
		}
	}
	return nil
}

//StopAdam function stop adam container
func StopAdam(adamRm bool) (err error) {
	state, err := utils.StateContainer(defaults.DefaultAdamContainerName)
	if err != nil {
		return fmt.Errorf("StopAdam: error in get state of adam container: %s", err)
	}
	if !strings.Contains(state, "running") {
		if adamRm {
			if err := utils.StopContainer(defaults.DefaultAdamContainerName, true); err != nil {
				return fmt.Errorf("StopAdam: error in rm adam container: %s", err)
			}
		}
	} else if state == "" {
		return nil
	} else {
		if adamRm {
			if err := utils.StopContainer(defaults.DefaultAdamContainerName, false); err != nil {
				return fmt.Errorf("StopAdam: error in rm adam container: %s", err)
			}
		} else {
			if err := utils.StopContainer(defaults.DefaultAdamContainerName, true); err != nil {
				return fmt.Errorf("StopAdam: error in rm adam container: %s", err)
			}
		}
	}
	return nil
}

//StatusAdam function return status of adam
func StatusAdam() (status string, err error) {
	state, err := utils.StateContainer(defaults.DefaultAdamContainerName)
	if err != nil {
		return "", fmt.Errorf("StatusAdam: error in get state of adam container: %s", err)
	}
	if state == "" {
		return "container doesn't exist", nil
	}
	return state, nil
}

//StartRegistry function run registry in docker
func StartRegistry(port int, tag, registryPath string, opts ...string) (err error) {
	containerName := defaults.DefaultRegistryContainerName
	ref := defaults.DefaultRegistryContainerRef
	serviceName := "registry"
	portMap := map[string]string{"5000": strconv.Itoa(port)}
	cmd := []string{}
	cmd = append(cmd, opts...)
	volumeMap := map[string]string{"/var/lib/registry": registryPath}
	state, err := utils.StateContainer(containerName)
	if err != nil {
		return fmt.Errorf("StartRegistry: error in get state of %s container: %s", serviceName, err)
	}
	if state == "" {
		if err := utils.CreateAndRunContainer(containerName, ref+":"+tag, portMap, volumeMap, cmd, nil); err != nil {
			return fmt.Errorf("StartRegistry: error in create %s container: %s", serviceName, err)
		}
	} else if !strings.Contains(state, "running") {
		if err := utils.StartContainer(containerName); err != nil {
			return fmt.Errorf("StartRegistry: error in restart %s container: %s", serviceName, err)
		}
	}
	return nil
}

// StopRegistry function stop registry container
func StopRegistry(rm bool) (err error) {
	containerName := defaults.DefaultRegistryContainerName
	serviceName := "registry"
	state, err := utils.StateContainer(containerName)
	if err != nil {
		return fmt.Errorf("StopRegistry: error in get state of %s container: %s", serviceName, err)
	}
	if !strings.Contains(state, "running") {
		if rm {
			if err := utils.StopContainer(containerName, true); err != nil {
				return fmt.Errorf("StopRegistry: error in rm %s container: %s", serviceName, err)
			}
		}
	} else if state == "" {
		return nil
	} else {
		if rm {
			if err := utils.StopContainer(containerName, false); err != nil {
				return fmt.Errorf("StopRegistry: error in rm %s container: %s", serviceName, err)
			}
		} else {
			if err := utils.StopContainer(containerName, true); err != nil {
				return fmt.Errorf("StopRegistry: error in rm %s container: %s", serviceName, err)
			}
		}
	}
	return nil
}

// StatusRegistry function return status of registry
func StatusRegistry() (status string, err error) {
	containerName := defaults.DefaultRegistryContainerName
	serviceName := "registry"
	state, err := utils.StateContainer(containerName)
	if err != nil {
		return "", fmt.Errorf("StatusRegistry: error in get state of %s container: %s", serviceName, err)
	}
	if state == "" {
		return "container doesn't exist", nil
	}
	return state, nil
}

//StartEServer function run eserver in docker
//if eserverForce is set, it recreates container
func StartEServer(serverPort int, imageDist string, eserverForce bool, eserverTag string) (err error) {
	portMap := map[string]string{"8888": strconv.Itoa(serverPort)}
	volumeMap := map[string]string{"/eserver/run/eserver/": imageDist}
	eserverServerCommand := strings.Fields("server")
	// lets make sure eserverImageDist exists
	if imageDist != "" && os.MkdirAll(imageDist, os.ModePerm) != nil {
		return fmt.Errorf("StartEServer: %s does not exist and can not be created", imageDist)
	}
	if eserverForce {
		_ = utils.StopContainer(defaults.DefaultEServerContainerName, true)
		if err := utils.CreateAndRunContainer(defaults.DefaultEServerContainerName, defaults.DefaultEServerContainerRef+":"+eserverTag, portMap, volumeMap, eserverServerCommand, nil); err != nil {
			return fmt.Errorf("StartEServer: error in create eserver container: %s", err)
		}
	} else {
		state, err := utils.StateContainer(defaults.DefaultEServerContainerName)
		if err != nil {
			return fmt.Errorf("StartEServer: error in get state of eserver container: %s", err)
		}
		if state == "" {
			if err := utils.CreateAndRunContainer(defaults.DefaultEServerContainerName, defaults.DefaultEServerContainerRef+":"+eserverTag, portMap, volumeMap, eserverServerCommand, nil); err != nil {
				return fmt.Errorf("StartEServer: error in create eserver container: %s", err)
			}
		} else if !strings.Contains(state, "running") {
			if err := utils.StartContainer(defaults.DefaultEServerContainerName); err != nil {
				return fmt.Errorf("StartEServer: error in restart eserver container: %s", err)
			}
		}
	}
	return nil
}

//StopEServer function stop eserver container
func StopEServer(eserverRm bool) (err error) {
	state, err := utils.StateContainer(defaults.DefaultEServerContainerName)
	if err != nil {
		return fmt.Errorf("StopEServer: error in get state of eserver container: %s", err)
	}
	if !strings.Contains(state, "running") {
		if eserverRm {
			if err := utils.StopContainer(defaults.DefaultEServerContainerName, true); err != nil {
				return fmt.Errorf("StopEServer: error in rm eserver container: %s", err)
			}
		}
	} else if state == "" {
		return nil
	} else {
		if eserverRm {
			if err := utils.StopContainer(defaults.DefaultEServerContainerName, false); err != nil {
				return fmt.Errorf("StopEServer: error in rm eserver container: %s", err)
			}
		} else {
			if err := utils.StopContainer(defaults.DefaultEServerContainerName, true); err != nil {
				return fmt.Errorf("StopEServer: error in rm eserver container: %s", err)
			}
		}
	}
	return nil
}

//StatusEServer function return eserver of adam
func StatusEServer() (status string, err error) {
	state, err := utils.StateContainer(defaults.DefaultEServerContainerName)
	if err != nil {
		return "", fmt.Errorf("StatusEServer: error in get eserver of adam container: %s", err)
	}
	if state == "" {
		return "container doesn't exist", nil
	}
	return state, nil
}

//GenerateEveCerts function generates certs for EVE
func GenerateEveCerts(certsDir, domain, ip, eveIP, uuid, devModel, ssid, password string, grubOptions []string, apiV1 bool) (err error) {
	model, err := models.GetDevModelByName(devModel)
	if err != nil {
		return fmt.Errorf("GenerateEveCerts: %s", err)
	}
	if _, err := os.Stat(certsDir); os.IsNotExist(err) {
		if err = os.MkdirAll(certsDir, 0755); err != nil {
			return fmt.Errorf("GenerateEveCerts: %s", err)
		}
	}
	edenHome, err := utils.DefaultEdenDir()
	if err != nil {
		return fmt.Errorf("GenerateEveCerts: %s", err)
	}
	globalCertsDir := filepath.Join(edenHome, defaults.DefaultCertsDist)
	if _, err := os.Stat(globalCertsDir); os.IsNotExist(err) {
		if err = os.MkdirAll(globalCertsDir, 0755); err != nil {
			return fmt.Errorf("GenerateEveCerts: %s", err)
		}
	}
	log.Debug("generating CA")
	caCertPath := filepath.Join(globalCertsDir, "root-certificate.pem")
	caKeyPath := filepath.Join(globalCertsDir, "root-certificate-key.pem")
	rootCert, rootKey := utils.GenCARoot()
	if _, err := tls.LoadX509KeyPair(caCertPath, caKeyPath); err == nil { //existing certs looks ok
		log.Info("Use existing certs")
		rootCert, err = utils.ParseCertificate(caCertPath)
		if err != nil {
			return fmt.Errorf("GenerateEveCerts: cannot parse certificate from %s: %s", caCertPath, err)
		}
		rootKey, err = utils.ParsePrivateKey(caKeyPath)
		if err != nil {
			return fmt.Errorf("GenerateEveCerts: cannot parse key from %s: %s", caKeyPath, err)
		}
	}
	if err := utils.WriteToFiles(rootCert, rootKey, caCertPath, caKeyPath); err != nil {
		return fmt.Errorf("GenerateEveCerts: %s", err)
	}
	serverCertPath := filepath.Join(globalCertsDir, "server.pem")
	serverKeyPath := filepath.Join(globalCertsDir, "server-key.pem")
	if _, err := tls.LoadX509KeyPair(serverCertPath, serverKeyPath); err != nil {
		log.Debug("generating Adam cert and key")
		ips := []net.IP{net.ParseIP(ip), net.ParseIP(eveIP), net.ParseIP("127.0.0.1")}
		ServerCert, ServerKey := utils.GenServerCertElliptic(rootCert, rootKey, big.NewInt(1), ips, []string{domain}, domain)
		if err := utils.WriteToFiles(ServerCert, ServerKey, serverCertPath, serverKeyPath); err != nil {
			return fmt.Errorf("GenerateEveCerts: %s", err)
		}
	}
	if !apiV1 {
		signingCertPath := filepath.Join(globalCertsDir, "signing.pem")
		signingKeyPath := filepath.Join(globalCertsDir, "signing-key.pem")
		if _, err := tls.LoadX509KeyPair(signingCertPath, signingKeyPath); err != nil {
			log.Debug("generating Adam signing cert and key")
			ips := []net.IP{net.ParseIP(ip), net.ParseIP(eveIP), net.ParseIP("127.0.0.1")}
			signingCert, signingKey := utils.GenServerCertElliptic(rootCert, rootKey, big.NewInt(1), ips, []string{domain}, domain)
			if err := utils.WriteToFiles(signingCert, signingKey, signingCertPath, signingKeyPath); err != nil {
				return fmt.Errorf("GenerateEveCerts signing: %s", err)
			}
		}
		encryptCertPath := filepath.Join(globalCertsDir, "encrypt.pem")
		encryptKeyPath := filepath.Join(globalCertsDir, "encrypt-key.pem")
		if _, err := tls.LoadX509KeyPair(encryptCertPath, encryptKeyPath); err != nil {
			log.Debug("generating Adam encrypt cert and key")
			ips := []net.IP{net.ParseIP(ip), net.ParseIP(eveIP), net.ParseIP("127.0.0.1")}
			encryptCert, encryptKey := utils.GenServerCertElliptic(rootCert, rootKey, big.NewInt(1), ips, []string{domain}, domain)
			if err := utils.WriteToFiles(encryptCert, encryptKey, encryptCertPath, encryptKeyPath); err != nil {
				return fmt.Errorf("GenerateEveCerts signing: %s", err)
			}
		}
	}
	log.Debug("generating EVE cert and key")
	if err := utils.CopyFile(caCertPath, filepath.Join(certsDir, "root-certificate.pem")); err != nil {
		return fmt.Errorf("GenerateEveCerts: %s", err)
	}
	// generate v2tlsbaseroot-certificates.pem as concatenation of default certificate and root-certificate
	certOut, err := os.Create(filepath.Join(certsDir, "v2tlsbaseroot-certificates.pem"))
	if err != nil {
		return err
	}
	if _, err := io.WriteString(certOut, defaults.V2TLS); err != nil {
		return err
	}
	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: rootCert.Raw}); err != nil {
		return err
	}
	if err := certOut.Close(); err != nil {
		return err
	}
	ClientCert, ClientKey := utils.GenServerCertElliptic(rootCert, rootKey, big.NewInt(2), nil, nil, uuid)
	log.Debug("saving files")
	if err := utils.WriteToFiles(ClientCert, ClientKey, filepath.Join(certsDir, "onboard.cert.pem"), filepath.Join(certsDir, "onboard.key.pem")); err != nil {
		return fmt.Errorf("GenerateEveCerts: %s", err)
	}
	log.Debug("generating ssh pair")
	if err := utils.GenerateSSHKeyPair(filepath.Join(certsDir, "id_rsa"), filepath.Join(certsDir, "id_rsa.pub")); err != nil {
		return fmt.Errorf("GenerateEveCerts: %s", err)
	}
	if ssid != "" && password != "" {
		log.Debug("generating DevicePortConfig")
		if portConfig := model.GetPortConfig(ssid, password); portConfig != "" {
			if _, err := os.Stat(filepath.Join(certsDir, "DevicePortConfig", "override.json")); os.IsNotExist(err) {
				if err := os.MkdirAll(filepath.Join(certsDir, "DevicePortConfig"), 0755); err != nil {
					return fmt.Errorf("GenerateEveCerts: %s", err)
				}
				if err := ioutil.WriteFile(filepath.Join(certsDir, "DevicePortConfig", "override.json"), []byte(portConfig), 0666); err != nil {
					return fmt.Errorf("GenerateEveCerts: %s", err)
				}
			}
		}
	}
	if model.DevModelType() == defaults.DefaultQemuModel && viper.GetString("eve.arch") == "arm64" {
		// we need to properly set console for qemu arm64
		grubOptions = append(grubOptions, "set_global dom0_console \"console=ttyAMA0,115200 $dom0_console\"")
	}
	if len(grubOptions) > 0 {
		f, err := os.Create(filepath.Join(certsDir, "grub.cfg"))
		if err != nil {
			return fmt.Errorf("GenerateEveCerts: cannot create grub file: %s", err)
		}
		defer f.Close()
		for _, line := range grubOptions {
			_, err = f.WriteString(line)
			if err != nil {
				return fmt.Errorf("GenerateEveCerts: cannot write to grub file: %s", err)
			}
		}
	}
	redisPasswordFile := filepath.Join(globalCertsDir, defaults.DefaultRedisPasswordFile)
	if _, err := os.Stat(redisPasswordFile); os.IsNotExist(err) {
		pwd := utils.GeneratePassword(8)
		if err := ioutil.WriteFile(redisPasswordFile, []byte(pwd), 0755); err != nil {
			return err
		}
	}
	return nil
}

//PutEveCerts function put certs for zedcontrol
func PutEveCerts(certsDir, devModel, ssid, password string) (err error) {
	model, err := models.GetDevModelByName(devModel)
	if err != nil {
		return fmt.Errorf("GenerateEveCerts: %s", err)
	}
	if _, err := os.Stat(certsDir); os.IsNotExist(err) {
		if err = os.MkdirAll(certsDir, 0755); err != nil {
			return fmt.Errorf("GenerateEveCerts: %s", err)
		}
	}
	edenHome, err := utils.DefaultEdenDir()
	if err != nil {
		return fmt.Errorf("GenerateEveCerts: %s", err)
	}
	globalCertsDir := filepath.Join(edenHome, defaults.DefaultCertsDist)
	if _, err := os.Stat(globalCertsDir); os.IsNotExist(err) {
		if err = os.MkdirAll(globalCertsDir, 0755); err != nil {
			return fmt.Errorf("GenerateEveCerts: %s", err)
		}
	}
	log.Debug("generating CA")
	caCertPath := filepath.Join(globalCertsDir, "root-certificate.pem")
	caKeyPath := filepath.Join(globalCertsDir, "root-certificate-key.pem")
	rootCert, rootKey := utils.GenCARoot()
	if _, err := tls.LoadX509KeyPair(caCertPath, caKeyPath); err == nil { //existing certs looks ok
		log.Info("Use existing certs")
		rootCert, err = utils.ParseCertificate(caCertPath)
		if err != nil {
			return fmt.Errorf("GenerateEveCerts: cannot parse certificate from %s: %s", caCertPath, err)
		}
		rootKey, err = utils.ParsePrivateKey(caKeyPath)
		if err != nil {
			return fmt.Errorf("GenerateEveCerts: cannot parse key from %s: %s", caKeyPath, err)
		}
	}
	if err := utils.WriteToFiles(rootCert, rootKey, caCertPath, caKeyPath); err != nil {
		return fmt.Errorf("GenerateEveCerts: %s", err)
	}
	log.Debug("locating EVE cert and key")
	if err := ioutil.WriteFile(filepath.Join(certsDir, "root-certificate.pem"), []byte(defaults.RootCert), 0600); err != nil {
		return fmt.Errorf("GenerateEveCerts: %s", err)
	}
	if err := ioutil.WriteFile(filepath.Join(certsDir, "onboard.cert.pem"), []byte(defaults.OnboardCert), 0600); err != nil {
		return fmt.Errorf("GenerateEveCerts: %s", err)
	}
	if err := ioutil.WriteFile(filepath.Join(certsDir, "onboard.key.pem"), []byte(defaults.OnboardKey), 0600); err != nil {
		return fmt.Errorf("GenerateEveCerts: %s", err)
	}
	if err := ioutil.WriteFile(filepath.Join(certsDir, "v2tlsbaseroot-certificates.pem"), []byte(defaults.V2TLS), 0600); err != nil {
		return fmt.Errorf("GenerateEveCerts: %s", err)
	}
	log.Debug("generating ssh pair")
	if err := utils.GenerateSSHKeyPair(filepath.Join(certsDir, "id_rsa"), filepath.Join(certsDir, "id_rsa.pub")); err != nil {
		return fmt.Errorf("GenerateEveCerts: %s", err)
	}
	if ssid != "" && password != "" {
		log.Debug("generating DevicePortConfig")
		if portConfig := model.GetPortConfig(ssid, password); portConfig != "" {
			if _, err := os.Stat(filepath.Join(certsDir, "DevicePortConfig", "override.json")); os.IsNotExist(err) {
				if err := os.MkdirAll(filepath.Join(certsDir, "DevicePortConfig"), 0755); err != nil {
					return fmt.Errorf("GenerateEveCerts: %s", err)
				}
				if err := ioutil.WriteFile(filepath.Join(certsDir, "DevicePortConfig", "override.json"), []byte(portConfig), 0666); err != nil {
					return fmt.Errorf("GenerateEveCerts: %s", err)
				}
			}
		}
	}
	if model.DevModelType() == defaults.DefaultQemuModel && viper.GetString("eve.arch") == "arm64" {
		// we need to properly set console for qemu arm64
		if err := ioutil.WriteFile(filepath.Join(certsDir, "grub.cfg"), []byte("set_global dom0_console \"console=ttyAMA0,115200 $dom0_console\""), 0666); err != nil {
			return fmt.Errorf("GenerateEveCerts: %s", err)
		}
	}
	redisPasswordFile := filepath.Join(globalCertsDir, defaults.DefaultRedisPasswordFile)
	if _, err := os.Stat(redisPasswordFile); os.IsNotExist(err) {
		pwd := utils.GeneratePassword(8)
		if err := ioutil.WriteFile(redisPasswordFile, []byte(pwd), 0755); err != nil {
			return err
		}
	}
	return nil
}

//GenerateEVEConfig function copy certs to EVE config folder
//if ip is empty will not fill hosts file
func GenerateEVEConfig(devModel, eveConfig string, domain string, ip string, port int,
	apiV1 bool, softserial string) (err error) {
	if _, err = os.Stat(eveConfig); os.IsNotExist(err) {
		if err = os.MkdirAll(eveConfig, 0755); err != nil {
			return fmt.Errorf("GenerateEVEConfig: %s", err)
		}
	}
	if apiV1 {
		if _, err = os.Stat(filepath.Join(eveConfig, "Force-API-V1")); os.IsNotExist(err) {
			if err := utils.TouchFile(filepath.Join(eveConfig, "Force-API-V1")); err != nil {
				return fmt.Errorf("GenerateEVEConfig: %s", err)
			}
		}
	}
	if ip != "" {
		if devModel != defaults.DefaultQemuModel {
			// Without SDN there is no DNS server that can translate adam's domain name.
			// Put static entry to /config/hosts.
			if _, err = os.Stat(filepath.Join(eveConfig, "hosts")); os.IsNotExist(err) {
				if err = ioutil.WriteFile(filepath.Join(eveConfig, "hosts"), []byte(fmt.Sprintf("%s %s\n", ip, domain)), 0666); err != nil {
					return fmt.Errorf("GenerateEVEConfig: %s", err)
				}
			}
		}
		if _, err = os.Stat(filepath.Join(eveConfig, "server")); os.IsNotExist(err) {
			if err = ioutil.WriteFile(filepath.Join(eveConfig, "server"), []byte(fmt.Sprintf("%s:%d\n", domain, port)), 0666); err != nil {
				return fmt.Errorf("GenerateEVEConfig: %s", err)
			}
		}
	} else {
		if _, err = os.Stat(filepath.Join(eveConfig, "server")); os.IsNotExist(err) {
			if err = ioutil.WriteFile(filepath.Join(eveConfig, "server"), []byte(domain), 0666); err != nil {
				return fmt.Errorf("GenerateEVEConfig: %s", err)
			}
		}
	}
	if softserial != "" {
		if err := ioutil.WriteFile(filepath.Join(eveConfig, "soft_serial"), []byte(softserial), 0666); err != nil {
			return fmt.Errorf("GenerateEVEConfig: %s", err)
		}
	}
	return nil
}

//CloneFromGit function clone from git into dist
func CloneFromGit(dist string, gitRepo string, tag string) (err error) {
	if _, err := os.Stat(dist); !os.IsNotExist(err) {
		return fmt.Errorf("CloneFromGit: directory already exists: %s", dist)
	}
	if tag == "" {
		tag = "master"
	}
	commandArgsString := fmt.Sprintf("clone --branch %s --depth 1 --single-branch %s %s", tag, gitRepo, dist)
	log.Infof("CloneFromGit run: %s %s", "git", commandArgsString)
	return utils.RunCommandWithLogAndWait("git", defaults.DefaultLogLevelToPrint, strings.Fields(commandArgsString)...)
}

//MakeEveInRepo build image of EVE from source
func MakeEveInRepo(desc utils.EVEDescription, dist string) (image, additional string, err error) {
	if _, err := os.Stat(dist); os.IsNotExist(err) {
		return "", "", fmt.Errorf("MakeEveInRepo: directory not exists: %s", dist)
	}
	if _, err := os.Stat(desc.ConfigPath); os.IsNotExist(err) {
		if err = os.MkdirAll(desc.ConfigPath, 0755); err != nil {
			return "", "", fmt.Errorf("MakeEveInRepo: %s", err)
		}
	}
	if desc.Arch == runtime.GOARCH {
		commandArgsString := fmt.Sprintf("-C %s pkgs", dist)
		log.Infof("MakeEveInRepo run: %s %s", "make", commandArgsString)
		err = utils.RunCommandWithLogAndWait("make", defaults.DefaultLogLevelToPrint, strings.Fields(commandArgsString)...)
	} else {
		log.Warnf("current arch (%s) is not equal target (%s), we do not support cross-builds now", runtime.GOARCH, desc.Arch)
	}
	image = filepath.Join(dist, "dist", desc.Arch, "current", fmt.Sprintf("live.%s", desc.Format))
	if desc.Format == "gcp" {
		image = filepath.Join(dist, "dist", desc.Arch, "current", "live.img.tar.gz")
	}
	commandArgsString := fmt.Sprintf("-C %s ZARCH=%s HV=%s CONF_DIR=%s IMG_FORMAT=%s MEDIA_SIZE=%d live",
		dist, desc.Arch, desc.HV, desc.ConfigPath, desc.Format, desc.ImageSizeMB)
	log.Infof("MakeEveInRepo run: %s %s", "make", commandArgsString)
	if err = utils.RunCommandWithLogAndWait("make", defaults.DefaultLogLevelToPrint, strings.Fields(commandArgsString)...); err != nil {
		log.Info(err)
	}
	switch desc.Arch {
	case "amd64":
		biosPath1 := filepath.Join(dist, "dist", desc.Arch, "current", "installer", "firmware", "OVMF.fd")
		biosPath2 := filepath.Join(dist, "dist", desc.Arch, "current", "installer", "firmware", "OVMF_CODE.fd")
		biosPath3 := filepath.Join(dist, "dist", desc.Arch, "current", "installer", "firmware", "OVMF_VARS.fd")
		additional = strings.Join([]string{biosPath1, biosPath2, biosPath3}, ",")
	case "arm64":
		biosPath1 := filepath.Join(dist, "dist", desc.Arch, "current", "installer", "firmware", "OVMF.fd")
		biosPath2 := filepath.Join(dist, "dist", desc.Arch, "current", "installer", "firmware", "OVMF_VARS.fd")
		additional = strings.Join([]string{biosPath1, biosPath2}, ",")
	default:
		return "", "", fmt.Errorf("MakeEveInRepo: unsupported arch %s", desc.Arch)
	}
	return
}

//CleanContext cleanup only context data
func CleanContext(eveDist, certsDist, imagesDist, evePID, eveUUID, vmName string, configSaved string, remote bool) (err error) {
	edenDir, err := utils.DefaultEdenDir()
	if err != nil {
		return fmt.Errorf("CleanContext: %s", err)
	}
	eveStatusFile := filepath.Join(edenDir, fmt.Sprintf("state-%s.yml", eveUUID))
	if _, err = os.Stat(eveStatusFile); !os.IsNotExist(err) {
		ctrl, err := controller.CloudPrepare()
		if err != nil {
			return fmt.Errorf("CleanContext: error in CloudPrepare: %s", err)
		}
		log.Debugf("Get devUUID for onboardUUID %s", eveUUID)
		devUUID, err := ctrl.DeviceGetByOnboardUUID(eveUUID)
		if err != nil {
			return fmt.Errorf("CleanContext: %s", err)
		}
		log.Debugf("Deleting devUUID %s", devUUID)
		if err := ctrl.DeviceRemove(devUUID); err != nil {
			return fmt.Errorf("CleanContext: %s", err)
		}
		log.Debugf("Deleting onboardUUID %s", eveUUID)
		if err := ctrl.OnboardRemove(eveUUID); err != nil {
			return fmt.Errorf("CleanContext: %s", err)
		}
		localViper := viper.New()
		localViper.SetConfigFile(eveStatusFile)
		if err := localViper.ReadInConfig(); err != nil {
			log.Debug(err)
		} else {
			eveConfigFile := localViper.GetString("eve-config")
			if _, err = os.Stat(eveConfigFile); !os.IsNotExist(err) {
				if err := os.Remove(eveConfigFile); err != nil {
					log.Debug(err)
				}
			}
		}
		if err = os.RemoveAll(eveStatusFile); err != nil {
			return fmt.Errorf("CleanContext: error in %s delete: %s", eveStatusFile, err)
		}
	}
	if !remote {
		if viper.GetString("eve.devModel") == defaults.DefaultVBoxModel {
			if err := StopEVEVBox(vmName); err != nil {
				log.Infof("cannot stop EVE: %s", err)
			} else {
				log.Infof("EVE stopped")
			}
			if err := DeleteEVEVBox(vmName); err != nil {
				log.Infof("cannot delete EVE: %s", err)
			}
		} else if viper.GetString("eve.devModel") == defaults.DefaultParallelsModel {
			if err := StopEVEParallels(vmName); err != nil {
				log.Infof("cannot stop EVE: %s", err)
			} else {
				log.Infof("EVE stopped")
			}
			if err := DeleteEVEParallels(vmName); err != nil {
				log.Infof("cannot delete EVE: %s", err)
			}
		} else {
			if err := StopEVEQemu(evePID); err != nil {
				log.Infof("cannot stop EVE: %s", err)
			} else {
				log.Infof("EVE stopped")
			}
			err := StopSWTPM(filepath.Join(imagesDist, "swtpm"))
			if err != nil {
				log.Errorf("cannot stop swtpm: %s", err)
			} else {
				log.Infof("swtpm is stopping")
			}
		}
	}
	if _, err = os.Stat(eveDist); !os.IsNotExist(err) {
		if err = os.RemoveAll(eveDist); err != nil {
			return fmt.Errorf("CleanContext: error in %s delete: %s", eveDist, err)
		}
	}
	if _, err = os.Stat(certsDist); !os.IsNotExist(err) {
		if err = os.RemoveAll(certsDist); err != nil {
			return fmt.Errorf("CleanContext: error in %s delete: %s", certsDist, err)
		}
	}
	if _, err = os.Stat(imagesDist); !os.IsNotExist(err) {
		if err = os.RemoveAll(imagesDist); err != nil {
			return fmt.Errorf("CleanContext: error in %s delete: %s", imagesDist, err)
		}
	}
	if _, err = os.Stat(configSaved); !os.IsNotExist(err) {
		if err = os.RemoveAll(configSaved); err != nil {
			return fmt.Errorf("CleanContext: error in %s delete: %s", configSaved, err)
		}
	}
	return nil
}

//StopEden teardown Eden
func StopEden(adamRm, redisRm, registryRm, eserverRm, eveRemote bool,
	evePidFile, swtpmPidFile, sdnPidFile, devModel, vmName string) {
	if err := StopAdam(adamRm); err != nil {
		log.Infof("cannot stop adam: %s", err)
	} else {
		log.Infof("adam stopped")
	}
	if err := StopRedis(redisRm); err != nil {
		log.Infof("cannot stop redis: %s", err)
	} else {
		log.Infof("redis stopped")
	}
	if err := StopRegistry(registryRm); err != nil {
		log.Infof("cannot stop registry: %s", err)
	} else {
		log.Infof("registry stopped")
	}
	if err := StopEServer(eserverRm); err != nil {
		log.Infof("cannot stop eserver: %s", err)
	} else {
		log.Infof("eserver stopped")
	}
	if !eveRemote {
		StopEve(evePidFile, swtpmPidFile, sdnPidFile, devModel, vmName)
	}
}

// StopEve stops EVE, vTPM and SDN.
func StopEve(evePidFile, swtpmPidFile, sdnPidFile, devModel, vmName string) {
	if devModel == defaults.DefaultVBoxModel {
		if err := StopEVEVBox(vmName); err != nil {
			log.Infof("cannot stop EVE: %s", err)
		} else {
			log.Infof("EVE stopped")
		}
	} else if devModel == defaults.DefaultParallelsModel {
		if err := StopEVEParallels(vmName); err != nil {
			log.Infof("cannot stop EVE: %s", err)
		} else {
			log.Infof("EVE stopped")
		}
	} else {
		if err := StopEVEQemu(evePidFile); err != nil {
			log.Infof("cannot stop EVE: %s", err)
		} else {
			log.Infof("EVE stopped")
		}
		if swtpmPidFile != "" {
			err := StopSWTPM(filepath.Dir(swtpmPidFile))
			if err != nil {
				log.Infof("cannot stop swtpm: %s", err)
			} else {
				log.Infof("swtpm is stopping")
			}
		}
		sdnConfig := edensdn.SdnVMConfig{
			PidFile: sdnPidFile,
			// Nothing else needed to stop the VM.
		}
		sdnVmRunner, err := edensdn.GetSdnVMRunner(devModel, sdnConfig)
		if err != nil {
			log.Fatalf("failed to get SDN VM runner: %v", err)
		}
		err = sdnVmRunner.Stop()
		if err != nil {
			log.Errorf("cannot stop SDN: %v", err)
		} else {
			log.Infof("SDN stopped")
		}
	}
}

//CleanEden teardown Eden and cleanup
func CleanEden(eveDist, adamDist, certsDist, imagesDist, eserverDist, redisDist,
	registryDist, configDir, evePID, sdnPID, configSaved string, remote bool,
	devModel, vmName string) (err error) {
	command := "swtpm"
	swtpmPidFile := filepath.Join(imagesDist, fmt.Sprintf("%s.pid", command))
	StopEden(true, true, true, true, remote,
		evePID, swtpmPidFile, sdnPID, devModel, vmName)
	if _, err = os.Stat(eveDist); !os.IsNotExist(err) {
		if err = os.RemoveAll(eveDist); err != nil {
			return fmt.Errorf("CleanEden: error in %s delete: %s", eveDist, err)
		}
	}
	if _, err = os.Stat(certsDist); !os.IsNotExist(err) {
		if err = os.RemoveAll(certsDist); err != nil {
			return fmt.Errorf("CleanEden: error in %s delete: %s", certsDist, err)
		}
	}
	if _, err = os.Stat(imagesDist); !os.IsNotExist(err) {
		if err = os.RemoveAll(imagesDist); err != nil {
			return fmt.Errorf("CleanEden: error in %s delete: %s", imagesDist, err)
		}
	}
	if _, err = os.Stat(eserverDist); !os.IsNotExist(err) {
		if err = os.RemoveAll(eserverDist); err != nil {
			return fmt.Errorf("CleanEden: error in %s delete: %s", eserverDist, err)
		}
	}
	if _, err = os.Stat(adamDist); !os.IsNotExist(err) {
		if err = os.RemoveAll(adamDist); err != nil {
			return fmt.Errorf("CleanEden: error in %s delete: %s", adamDist, err)
		}
	}
	if _, err = os.Stat(redisDist); !os.IsNotExist(err) {
		if err = os.RemoveAll(redisDist); err != nil {
			return fmt.Errorf("CleanEden: error in %s delete: %s", redisDist, err)
		}
	}
	if _, err = os.Stat(registryDist); !os.IsNotExist(err) {
		if err = os.RemoveAll(registryDist); err != nil {
			return fmt.Errorf("CleanEden: error in %s delete: %s", registryDist, err)
		}
	}
	if _, err = os.Stat(configDir); !os.IsNotExist(err) {
		if err = os.RemoveAll(configDir); err != nil {
			return fmt.Errorf("CleanEden: error in %s delete: %s", configDir, err)
		}
	}
	if _, err = os.Stat(configSaved); !os.IsNotExist(err) {
		if err = os.RemoveAll(configSaved); err != nil {
			return fmt.Errorf("CleanEden: error in %s delete: %s", configSaved, err)
		}
	}
	if err = utils.RemoveGeneratedVolumeOfContainer(defaults.DefaultEServerContainerName); err != nil {
		return fmt.Errorf("CleanEden: RemoveGeneratedVolumeOfContainer for %s: %s", defaults.DefaultEServerContainerName, err)
	}
	if err = utils.RemoveGeneratedVolumeOfContainer(defaults.DefaultRedisContainerName); err != nil {
		return fmt.Errorf("CleanEden: RemoveGeneratedVolumeOfContainer for %s: %s", defaults.DefaultRedisContainerName, err)
	}
	if err = utils.RemoveGeneratedVolumeOfContainer(defaults.DefaultAdamContainerName); err != nil {
		return fmt.Errorf("CleanEden: RemoveGeneratedVolumeOfContainer for %s: %s", defaults.DefaultAdamContainerName, err)
	}
	if err = utils.RemoveGeneratedVolumeOfContainer(defaults.DefaultRegistryContainerName); err != nil {
		return fmt.Errorf("CleanEden: RemoveGeneratedVolumeOfContainer for %s: %s", defaults.DefaultRegistryContainerName, err)
	}
	if devModel == defaults.DefaultVBoxModel {
		if err := DeleteEVEVBox(vmName); err != nil {
			log.Infof("cannot delete EVE: %s", err)
		}
	} else if devModel == defaults.DefaultParallelsModel {
		if err := DeleteEVEParallels(vmName); err != nil {
			log.Infof("cannot delete EVE: %s", err)
		}
	}
	return nil
}

//EServer for connection to eserver
type EServer struct {
	EServerIP   string
	EServerPort string
}

func (server *EServer) getHTTPClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			ResponseHeaderTimeout: defaults.DefaultRepeatTimeout * defaults.DefaultRepeatCount,
		},
	}
}

//EServerAddFileURL send url to download image into eserver
func (server *EServer) EServerAddFileURL(url string) (name string) {
	u, err := utils.ResolveURL(fmt.Sprintf("http://%s:%s", server.EServerIP, server.EServerPort), "admin/add-from-url")
	if err != nil {
		log.Fatalf("error constructing URL: %v", err)
	}
	client := server.getHTTPClient(defaults.DefaultRepeatTimeout)
	objToSend := api.URLArg{
		URL: url,
	}
	body, err := json.Marshal(objToSend)
	if err != nil {
		log.Fatalf("EServerAddFileURL: error encoding json: %v", err)
	}
	req, err := http.NewRequest("POST", u, bytes.NewBuffer(body))
	if err != nil {
		log.Fatalf("EServerAddFileURL: unable to create new http request: %v", err)
	}

	response, err := utils.RepeatableAttempt(client, req)
	if err != nil {
		log.Fatalf("EServerAddFileURL: unable to send request: %v", err)
	}
	buf, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Fatalf("EServerAddFileURL: unable to read data from URL %s: %v", u, err)
	}
	return string(buf)
}

//EServerCheckStatus checks status of image in eserver
func (server *EServer) EServerCheckStatus(name string) (fileInfo *api.FileInfo) {
	u, err := utils.ResolveURL(fmt.Sprintf("http://%s:%s", server.EServerIP, server.EServerPort), fmt.Sprintf("admin/status/%s", name))
	if err != nil {
		log.Fatalf("EServerAddFileURL: error constructing URL: %v", err)
	}
	client := server.getHTTPClient(defaults.DefaultRepeatTimeout * defaults.DefaultRepeatCount)
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		log.Fatalf("EServerAddFileURL: unable to create new http request: %v", err)
	}

	response, err := utils.RepeatableAttempt(client, req)
	if err != nil {
		log.Fatalf("EServerAddFileURL: unable to send request: %v", err)
	}
	buf, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Fatalf("EServerAddFileURL: unable to read data from URL %s: %v", u, err)
	}
	if err := json.Unmarshal(buf, &fileInfo); err != nil {
		log.Fatalf("EServerAddFileURL: %s", err)
	}
	return
}

//EServerAddFile send file with image into eserver
func (server *EServer) EServerAddFile(filepath, prefix string) (fileInfo *api.FileInfo) {
	u, err := utils.ResolveURL(fmt.Sprintf("http://%s:%s", server.EServerIP, server.EServerPort), "admin/add-from-file")
	if err != nil {
		log.Fatalf("EServerAddFile: error constructing URL: %v", err)
	}
	client := server.getHTTPClient(0)
	response, err := utils.UploadFile(client, u, filepath, prefix)
	if err != nil {
		log.Fatalf("EServerAddFile: %s", err)
	}
	buf, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Fatalf("EServerAddFile: unable to read data from URL %s: %v", u, err)
	}
	if err := json.Unmarshal(buf, &fileInfo); err != nil {
		log.Fatalf("EServerAddFile: %s", err)
	}
	return
}

//ReadFileInSquashFS returns the content of a single file (filePath) inside squashfs (squashFSPath)
func ReadFileInSquashFS(squashFSPath, filePath string) (content []byte, err error) {
	tmpdir, err := ioutil.TempDir("", "squashfs-unpack")
	if err != nil {
		return nil, fmt.Errorf("ReadFileInSquashFS: %s", err)
	}
	defer os.RemoveAll(tmpdir)
	dirToUnpack := filepath.Join(tmpdir, "temp")
	if output, err := exec.Command("unsquashfs", "-n", "-i", "-d", dirToUnpack, squashFSPath, filePath).CombinedOutput(); err != nil {
		return nil, fmt.Errorf("ReadFileInSquashFS: unsquashfs (%s): %v", output, err)
	}
	content, err = ioutil.ReadFile(filepath.Join(dirToUnpack, filePath))
	if err != nil {
		return nil, fmt.Errorf("ReadFileInSquashFS: %s", err)
	}
	return content, nil
}

//EVEInfo contains info from SD card
type EVEInfo struct {
	EVERelease []byte //EVERelease is /etc/eve-release from rootfs
	Syslog     []byte //Syslog is /rsyslog/syslog.txt from persist volume
}

//GetInfoFromSDCard obtain info from SD card with EVE
// devicePath is /dev/sdX or /dev/diskX
func GetInfoFromSDCard(devicePath string) (eveInfo *EVEInfo, err error) {
	eveInfo = &EVEInfo{}
	// /dev/sdXN notation by default
	rootfsDev := fmt.Sprintf("%s2", devicePath)
	persistDev := fmt.Sprintf("%s9", devicePath)
	// /dev/diskXsN notation for MacOS
	if runtime.GOOS == `darwin` {
		rootfsDev = fmt.Sprintf("%ss2", devicePath)
		persistDev = fmt.Sprintf("%ss9", devicePath)
	}
	if _, err := os.Stat(devicePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("GetInfoFromSDCard: %s", err)
	}
	if _, err := os.Stat(rootfsDev); !os.IsNotExist(err) {
		eveRelease, err := ReadFileInSquashFS(rootfsDev, "/etc/eve-release")
		if err != nil {
			return nil, fmt.Errorf("GetInfoFromSDCard: %s", err)
		}
		eveInfo.EVERelease = eveRelease
	}
	if _, err := os.Stat(persistDev); !os.IsNotExist(err) {
		fsPersist, err := gexto.NewFileSystem(persistDev)
		if err != nil {
			return nil, fmt.Errorf("GetInfoFromSDCard: %s", err)
		}
		g, err := fsPersist.Open("/rsyslog/syslog.txt")
		if err != nil {
			return nil, fmt.Errorf("GetInfoFromSDCard: %s", err)
		}
		syslog, err := ioutil.ReadAll(g)
		if err != nil {
			return nil, fmt.Errorf("GetInfoFromSDCard: %s", err)
		}
		eveInfo.Syslog = syslog
	}
	return eveInfo, nil
}

//AddFileIntoEServer puts file into eserver
//prefix will be added to the file if defined
func AddFileIntoEServer(server *EServer, filePath, prefix string) (*api.FileInfo, error) {
	fileName := filepath.Base(filePath)
	if prefix != "" {
		fileName = fmt.Sprintf("%s/%s", prefix, fileName)
	}
	status := server.EServerCheckStatus(fileName)
	if !status.ISReady || status.Size != utils.GetFileSize(filePath) || status.Sha256 != utils.SHA256SUM(filePath) {
		log.Infof("Start uploading into eserver of %s", filePath)
		status = server.EServerAddFile(filePath, prefix)
		if status.Error != "" {
			return nil, fmt.Errorf("AddFileIntoEServer: %s", status.Error)
		}
	}
	return status, nil
}
