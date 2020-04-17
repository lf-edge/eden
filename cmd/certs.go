package cmd

import (
	"github.com/lf-edge/eden/pkg/utils"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"math/big"
	"net"
	"os"
	"path"
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
	Run: func(cmd *cobra.Command, args []string) {
		var err error
		rootCert, rootKey := utils.GenCARoot()
		ServerCert, ServerKey := utils.GenServerCert(rootCert, rootKey, big.NewInt(1), []net.IP{net.ParseIP(certsIP)}, []string{certsDomain}, certsDomain)
		ClientCert, ClientKey := utils.GenServerCert(rootCert, rootKey, big.NewInt(2), nil, nil, certsUUID)
		err = utils.WriteToFiles(rootCert, rootKey, path.Join(certsDir, "root-certificate.pem"), path.Join(certsDir, "root-certificate.key"))
		if err != nil {
			log.Fatal(err)
		}
		err = utils.WriteToFiles(ServerCert, ServerKey, path.Join(certsDir, "server.pem"), path.Join(certsDir, "server-key.pem"))
		if err != nil {
			log.Fatal(err)
		}
		err = utils.WriteToFiles(ClientCert, ClientKey, path.Join(certsDir, "onboard.cert.pem"), path.Join(certsDir, "onboard.key.pem"))
		if err != nil {
			log.Fatal(err)
		}
	},
}

func certsInit() {
	currentPath, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	certsCmd.Flags().StringVarP(&certsDir, "output", "o", currentPath, "directory to save")
	certsCmd.Flags().StringVarP(&certsDomain, "domain", "d", defaultDomain, "FQDN for certificates")
	certsCmd.Flags().StringVarP(&certsIP, "ip", "i", defaultIP, "IP address to use")
	certsCmd.Flags().StringVarP(&certsUUID, "uuid", "u", defaultUUID, "UUID to use for device")
}
