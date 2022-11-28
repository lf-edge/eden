package cmd

import (
	"os"
	"path/filepath"

	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/eden"
	"github.com/lf-edge/eden/pkg/openevec"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func newCertsCmd(cfg *openevec.EdenSetupArgs) *cobra.Command {
	var certsCmd = &cobra.Command{
		Use:   "certs",
		Short: "manage certs",
		Long:  `Managed certificates for Adam and EVE.`,
		Run: func(cmd *cobra.Command, args []string) {
			if err := eden.GenerateEveCerts(cfg.Eden.CertsDir, cfg.Adam.CertsDomain, cfg.Adam.CertsIP, cfg.Adam.CertsEVEIP, cfg.Eve.CertsUUID, cfg.Eve.DevModel, cfg.Eve.Ssid, cfg.Eve.Password, cfg.Runtime.GrubOptions, cfg.Adam.APIv1); err != nil {
				log.Errorf("cannot GenerateEveCerts: %s", err)
			} else {
				log.Info("GenerateEveCerts done")
			}
		},
	}

	currentPath, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	certsCmd.Flags().StringVarP(&cfg.Adam.Tag, "adam-tag", "", defaults.DefaultAdamTag, "tag on adam container to pull")
	certsCmd.Flags().StringVarP(&cfg.Adam.Dist, "adam-dist", "", cfg.Adam.Dist, "adam dist to start (required)")
	certsCmd.Flags().IntVarP(&cfg.Adam.Port, "adam-port", "", defaults.DefaultAdamPort, "adam port to start")
	certsCmd.Flags().BoolVarP(&cfg.Adam.Force, "adam-force", "", cfg.Adam.Force, "adam force rebuild")
	certsCmd.Flags().StringVarP(&cfg.Eden.CertsDir, "certs-dist", "o", filepath.Join(currentPath, defaults.DefaultDist, defaults.DefaultCertsDist), "directory to save")
	certsCmd.Flags().StringVarP(&cfg.Adam.CertsDomain, "domain", "d", defaults.DefaultDomain, "FQDN for certificates")
	certsCmd.Flags().StringVarP(&cfg.Adam.CertsIP, "ip", "i", defaults.DefaultIP, "IP address to use")
	certsCmd.Flags().StringVarP(&cfg.Adam.CertsEVEIP, "eve-ip", "", defaults.DefaultEVEIP, "IP address to use for EVE")
	certsCmd.Flags().StringVarP(&cfg.Eve.CertsUUID, "uuid", "u", defaults.DefaultUUID, "UUID to use for device")
	certsCmd.Flags().StringVar(&cfg.Eve.Ssid, "ssid", "", "SSID for wifi")
	certsCmd.Flags().StringVar(&cfg.Eve.Password, "password", "", "password for wifi")
	certsCmd.Flags().StringArrayVar(&cfg.Runtime.GrubOptions, "grub-options", []string{}, "append lines to grub options")

	return certsCmd
}
