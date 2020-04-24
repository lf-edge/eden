package cmd

import (
	"fmt"
	"github.com/lf-edge/eden/pkg/utils"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"math/big"
	"net"
	"os"
	"path/filepath"
)

const (
	defaultDomain = "mydomain.adam"
	defaultIP     = "192.168.0.1"
	defaultUUID   = "1"
)

var (
	certsDir    string
	certsDomain string
	certsIP     string
	certsUUID   string
)

var certsCmd = &cobra.Command{
	Use:   "certs",
	Short: "manage certs",
	Long:  `Managed certificates for Adam and EVE.`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		viperLoaded, err := utils.LoadConfigFile(config)
		if err != nil {
			return fmt.Errorf("error reading config: %s", err.Error())
		}
		if viperLoaded {
			certsDir = viper.GetString("certs-dist")
			certsDomain = viper.GetString("domain")
			certsIP = viper.GetString("ip")
			certsUUID = viper.GetString("uuid")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		if _, err := os.Stat(certsDir); os.IsNotExist(err) {
			if err = os.MkdirAll(certsDir, 0755); err != nil {
				log.Fatal(err)
			}
		}
		rootCert, rootKey := utils.GenCARoot()
		ServerCert, ServerKey := utils.GenServerCert(rootCert, rootKey, big.NewInt(1), []net.IP{net.ParseIP(certsIP)}, []string{certsDomain}, certsDomain)
		ClientCert, ClientKey := utils.GenServerCert(rootCert, rootKey, big.NewInt(2), nil, nil, certsUUID)
		if err := utils.WriteToFiles(rootCert, rootKey, filepath.Join(certsDir, "root-certificate.pem"), filepath.Join(certsDir, "root-certificate.key")); err != nil {
			log.Fatal(err)
		}
		if err := utils.WriteToFiles(ServerCert, ServerKey, filepath.Join(certsDir, "server.pem"), filepath.Join(certsDir, "server-key.pem")); err != nil {
			log.Fatal(err)
		}
		if err := utils.WriteToFiles(ClientCert, ClientKey, filepath.Join(certsDir, "onboard.cert.pem"), filepath.Join(certsDir, "onboard.key.pem")); err != nil {
			log.Fatal(err)
		}
		if err := utils.GenerateSSHKeyPair(filepath.Join(certsDir, "id_rsa"), filepath.Join(certsDir, "id_rsa.pub")); err != nil {
			log.Fatal(err)
		}
	},
}

func certsInit() {
	currentPath, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	certsCmd.Flags().StringVarP(&certsDir, "certs-dist", "o", filepath.Join(currentPath, "dist", "certs"), "directory to save")
	certsCmd.Flags().StringVarP(&certsDomain, "domain", "d", defaultDomain, "FQDN for certificates")
	certsCmd.Flags().StringVarP(&certsIP, "ip", "i", defaultIP, "IP address to use")
	certsCmd.Flags().StringVarP(&certsUUID, "uuid", "u", defaultUUID, "UUID to use for device")
	if err := viper.BindPFlags(certsCmd.Flags()); err != nil {
		log.Fatal(err)
	}
	certsCmd.Flags().StringVar(&config, "config", "", "path to config file")
}
