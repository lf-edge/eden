package cmd

import (
	"fmt"
	"github.com/lf-edge/eden/pkg/controller"
	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/utils"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"io/ioutil"
	"math/big"
	"net"
	"os"
	"path/filepath"
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
		assingCobraToViper(cmd)
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
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		if _, err := os.Stat(certsDir); os.IsNotExist(err) {
			if err = os.MkdirAll(certsDir, 0755); err != nil {
				log.Fatal(err)
			}
		}
		log.Debug("generating CA")
		rootCert, rootKey := utils.GenCARoot()
		log.Debug("generating Adam cert and key")
		ips := []net.IP{net.ParseIP(certsIP), net.ParseIP(certsEVEIP), net.ParseIP("127.0.0.1")}
		ServerCert, ServerKey := utils.GenServerCert(rootCert, rootKey, big.NewInt(1), ips, []string{certsDomain}, certsDomain)
		log.Debug("generating EVE cert and key")
		ClientCert, ClientKey := utils.GenServerCert(rootCert, rootKey, big.NewInt(2), nil, nil, certsUUID)
		log.Debug("saving files")
		if err := utils.WriteToFiles(rootCert, rootKey, filepath.Join(certsDir, "root-certificate.pem"), filepath.Join(certsDir, "root-certificate.key")); err != nil {
			log.Fatal(err)
		}
		if err := utils.WriteToFiles(ServerCert, ServerKey, filepath.Join(certsDir, "server.pem"), filepath.Join(certsDir, "server-key.pem")); err != nil {
			log.Fatal(err)
		}
		if err := utils.WriteToFiles(ClientCert, ClientKey, filepath.Join(certsDir, "onboard.cert.pem"), filepath.Join(certsDir, "onboard.key.pem")); err != nil {
			log.Fatal(err)
		}
		log.Debug("generating ssh pair")
		if err := utils.GenerateSSHKeyPair(filepath.Join(certsDir, "id_rsa"), filepath.Join(certsDir, "id_rsa.pub")); err != nil {
			log.Fatal(err)
		}
		if ssid != "" && password != "" {
			log.Debug("generating DevicePortConfig")
			if portConfig := controller.GetPortConfig(devModel, ssid, password); portConfig != "" {
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

	certsCmd.Flags().StringVarP(&certsDir, "certs-dist", "o", filepath.Join(currentPath, defaults.DefaultDist, defaults.DefaultCertsDist), "directory to save")
	certsCmd.Flags().StringVarP(&certsDomain, "domain", "d", defaults.DefaultDomain, "FQDN for certificates")
	certsCmd.Flags().StringVarP(&certsIP, "ip", "i", defaults.DefaultIP, "IP address to use")
	certsCmd.Flags().StringVarP(&certsEVEIP, "eve-ip", "", defaults.DefaultEVEIP, "IP address to use for EVE")
	certsCmd.Flags().StringVarP(&certsUUID, "uuid", "u", defaults.DefaultUUID, "UUID to use for device")
	certsCmd.Flags().StringVar(&ssid, "ssid", "", "SSID for wifi")
	certsCmd.Flags().StringVar(&password, "password", "", "password for wifi")
}
