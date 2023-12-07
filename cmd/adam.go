package cmd

import (
	"fmt"
	"os"

	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/eden"
	"github.com/lf-edge/eden/pkg/openevec"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func newAdamCmd(configName, verbosity *string) *cobra.Command {
	cfg := &openevec.EdenSetupArgs{}
	var adamCmd = &cobra.Command{
		Use:               "adam",
		PersistentPreRunE: preRunViperLoadFunction(cfg, configName, verbosity),
	}

	groups := CommandGroups{
		{
			Message: "Basic Commands",
			Commands: []*cobra.Command{
				newStartAdamCmd(cfg),
				newStopAdamCmd(),
				newStatusAdamCmd(),
				newChangeCertCmd(),
			},
		},
	}

	groups.AddTo(adamCmd)

	return adamCmd
}

func newStartAdamCmd(cfg *openevec.EdenSetupArgs) *cobra.Command {
	var startAdamCmd = &cobra.Command{
		Use:   "start",
		Short: "start adam",
		Long:  `Start adam.`,
		Run: func(cmd *cobra.Command, args []string) {
			if err := openEVEC.AdamStart(); err != nil {
				log.Fatalf("Adam start failed: %s", err)
			}
		},
	}

	startAdamCmd.Flags().StringVarP(&cfg.Adam.Tag, "adam-tag", "", defaults.DefaultAdamTag, "tag on adam container to pull")
	startAdamCmd.Flags().StringVarP(&cfg.Adam.Dist, "adam-dist", "", cfg.Adam.Dist, "adam dist to start (required)")
	startAdamCmd.Flags().IntVarP(&cfg.Adam.Port, "adam-port", "", defaults.DefaultAdamPort, "adam port to start")
	startAdamCmd.Flags().BoolVarP(&cfg.Adam.Force, "adam-force", "", cfg.Adam.Force, "adam force rebuild")
	startAdamCmd.Flags().StringVarP(&cfg.Adam.Redis.RemoteURL, "adam-redis-url", "", cfg.Adam.Redis.RemoteURL, "adam remote redis url")
	startAdamCmd.Flags().BoolVarP(&cfg.Adam.Remote.Redis, "adam-redis", "", cfg.Adam.Remote.Redis, "use adam remote redis")

	return startAdamCmd
}

func newStopAdamCmd() *cobra.Command {
	var adamRm bool

	var stopAdamCmd = &cobra.Command{
		Use:   "stop",
		Short: "stop adam",
		Long:  `Stop adam.`,
		Run: func(cmd *cobra.Command, args []string) {
			if err := eden.StopAdam(adamRm); err != nil {
				log.Errorf("cannot stop adam: %s", err)
			}
		},
	}

	stopAdamCmd.Flags().BoolVarP(&adamRm, "adam-rm", "", false, "adam rm on stop")

	return stopAdamCmd
}

func newStatusAdamCmd() *cobra.Command {
	var statusAdamCmd = &cobra.Command{
		Use:   "status",
		Short: "status of adam",
		Long:  `Status of adam.`,
		Run: func(cmd *cobra.Command, args []string) {
			statusAdam, err := eden.StatusAdam()
			if err != nil {
				log.Errorf("cannot obtain status of adam: %s", err)
			} else {
				fmt.Printf("Adam status: %s\n", statusAdam)
			}
		},
	}

	return statusAdamCmd
}

func newChangeCertCmd() *cobra.Command {
	var certFile string

	var changeCertCmd = &cobra.Command{
		Use:   "change-signing-cert",
		Short: "change signing certificate for adam",
		Long:  `Set Adam's signing certificate from a file.`,
		Run: func(cmd *cobra.Command, args []string) {
			certData, err := os.ReadFile(certFile)
			if err != nil {
				log.Fatalf("Failed to read certificate file: %s", err)
			}

			if err := openEVEC.ChangeSigningCert(certData); err != nil {
				log.Fatalf("Failed to upload certificate to adam: %s", err)
			}
		},
	}

	changeCertCmd.Flags().StringVarP(&certFile, "cert-file", "", "", "path to the signing certificate file")

	return changeCertCmd
}
