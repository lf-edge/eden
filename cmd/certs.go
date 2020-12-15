package cmd

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"math/big"
	"net"
	"os"
	"path/filepath"

	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/models"
	"github.com/lf-edge/eden/pkg/utils"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	certsDir    string
	certsDomain string
	certsIP     string
	certsEVEIP  string
	certsUUID   string
)

var certsCmd = &cobra.Command{
	Use:   "certs",
	Short: "manage certs",
	Long:  `Managed certificates for Adam and EVE.`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		assignCobraToViper(cmd)
		viperLoaded, err := utils.LoadConfigFile(configFile)
		if err != nil {
			return fmt.Errorf("error reading config: %s", err.Error())
		}
		if viperLoaded {
			certsDir = utils.ResolveAbsPath(viper.GetString("eden.certs-dist"))
			certsDomain = viper.GetString("adam.domain")
			certsIP = viper.GetString("adam.ip")
			certsEVEIP = viper.GetString("adam.eve-ip")
			certsUUID = viper.GetString("eve.uuid")
			certsUUID = viper.GetString("eve.uuid")
			devModel = viper.GetString("eve.devmodel")
			adamTag = viper.GetString("adam.tag")
			adamPort = viper.GetInt("adam.port")
			adamDist = utils.ResolveAbsPath(viper.GetString("adam.dist"))
			adamForce = viper.GetBool("adam.force")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		model, err := models.GetDevModelByName(devModel)
		if err != nil {
			log.Fatalf("GetDevModelByName: %s", err)
		}
		if _, err := os.Stat(certsDir); os.IsNotExist(err) {
			if err = os.MkdirAll(certsDir, 0755); err != nil {
				log.Fatal(err)
			}
		}
		edenHome, err := utils.DefaultEdenDir()
		if err != nil {
			log.Fatal(err)
		}
		globalCertsDir := filepath.Join(edenHome, defaults.DefaultCertsDist)
		if _, err := os.Stat(globalCertsDir); os.IsNotExist(err) {
			if err = os.MkdirAll(globalCertsDir, 0755); err != nil {
				log.Fatal(err)
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
				log.Fatalf("cannot parse certificate from %s: %s", caCertPath, err)
			}
			rootKey, err = utils.ParsePrivateKey(caKeyPath)
			if err != nil {
				log.Fatalf("cannot parse key from %s: %s", caKeyPath, err)
			}
		}
		if err := utils.WriteToFiles(rootCert, rootKey, caCertPath, caKeyPath); err != nil {
			log.Fatal(err)
		}
		serverCertPath := filepath.Join(globalCertsDir, "server.pem")
		serverKeyPath := filepath.Join(globalCertsDir, "server-key.pem")
		if _, err := tls.LoadX509KeyPair(serverCertPath, serverKeyPath); err != nil {
			log.Debug("generating Adam cert and key")
			ips := []net.IP{net.ParseIP(certsIP), net.ParseIP(certsEVEIP), net.ParseIP("127.0.0.1")}
			ServerCert, ServerKey := utils.GenServerCert(rootCert, rootKey, big.NewInt(1), ips, []string{certsDomain}, certsDomain)
			if err := utils.WriteToFiles(ServerCert, ServerKey, serverCertPath, serverKeyPath); err != nil {
				log.Fatal(err)
			}
		}
		log.Debug("generating EVE cert and key")
		if err := utils.CopyFile(caCertPath, filepath.Join(certsDir, "root-certificate.pem")); err != nil {
			log.Fatal(err)
		}
		ClientCert, ClientKey := utils.GenServerCert(rootCert, rootKey, big.NewInt(2), nil, nil, certsUUID)
		log.Debug("saving files")
		if err := utils.WriteToFiles(ClientCert, ClientKey, filepath.Join(certsDir, "onboard.cert.pem"), filepath.Join(certsDir, "onboard.key.pem")); err != nil {
			log.Fatal(err)
		}
		log.Debug("generating ssh pair")
		if err := utils.GenerateSSHKeyPair(filepath.Join(certsDir, "id_rsa"), filepath.Join(certsDir, "id_rsa.pub")); err != nil {
			log.Fatal(err)
		}
		if ssid != "" && password != "" {
			log.Debug("generating DevicePortConfig")
			if portConfig := model.GetPortConfig(ssid, password); portConfig != "" {
				if _, err := os.Stat(filepath.Join(certsDir, "DevicePortConfig", "override.json")); os.IsNotExist(err) {
					if err := os.MkdirAll(filepath.Join(certsDir, "DevicePortConfig"), 0755); err != nil {
						log.Fatal(err)
					}
					if err := ioutil.WriteFile(filepath.Join(certsDir, "DevicePortConfig", "override.json"), []byte(portConfig), 0666); err != nil {
						log.Fatal(err)
					}
				}
			}
		}
	},
}

func certsInit() {
	currentPath, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	certsCmd.Flags().StringVarP(&adamTag, "adam-tag", "", defaults.DefaultAdamTag, "tag on adam container to pull")
	certsCmd.Flags().StringVarP(&adamDist, "adam-dist", "", "", "adam dist to start (required)")
	certsCmd.Flags().IntVarP(&adamPort, "adam-port", "", defaults.DefaultAdamPort, "adam port to start")
	certsCmd.Flags().BoolVarP(&adamForce, "adam-force", "", false, "adam force rebuild")
	certsCmd.Flags().StringVarP(&certsDir, "certs-dist", "o", filepath.Join(currentPath, defaults.DefaultDist, defaults.DefaultCertsDist), "directory to save")
	certsCmd.Flags().StringVarP(&certsDomain, "domain", "d", defaults.DefaultDomain, "FQDN for certificates")
	certsCmd.Flags().StringVarP(&certsIP, "ip", "i", defaults.DefaultIP, "IP address to use")
	certsCmd.Flags().StringVarP(&certsEVEIP, "eve-ip", "", defaults.DefaultEVEIP, "IP address to use for EVE")
	certsCmd.Flags().StringVarP(&certsUUID, "uuid", "u", defaults.DefaultUUID, "UUID to use for device")
	certsCmd.Flags().StringVar(&ssid, "ssid", "", "SSID for wifi")
	certsCmd.Flags().StringVar(&password, "password", "", "password for wifi")
}
