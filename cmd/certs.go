package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/eden"
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
			apiV1 = viper.GetBool("adam.v1")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		if err := eden.GenerateEveCerts(certsDir, certsDomain, certsIP, certsEVEIP, certsUUID, devModel, ssid, password, apiV1); err != nil {
			log.Errorf("cannot GenerateEveCerts: %s", err)
		} else {
			log.Info("GenerateEveCerts done")
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
