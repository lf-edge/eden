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

func newEserverCmd(configName, verbosity *string) *cobra.Command {
	cfg := &openevec.EdenSetupArgs{}
	var eserverCmd = &cobra.Command{
		Use:               "eserver",
		PersistentPreRunE: preRunViperLoadFunction(cfg, configName, verbosity),
	}

	groups := CommandGroups{
		{
			Message: "Basic Commands",
			Commands: []*cobra.Command{
				newStartEserverCmd(cfg),
				newStopEserverCmd(cfg),
				newStatusEserverCmd(cfg),
			},
		},
	}

	groups.AddTo(eserverCmd)

	return eserverCmd
}

func newStartEserverCmd(cfg *openevec.EdenSetupArgs) *cobra.Command {

	var startEserverCmd = &cobra.Command{
		Use:   "start",
		Short: "start eserver",
		Long:  `Start eserver.`,
		Run: func(cmd *cobra.Command, args []string) {
			command, err := os.Executable()
			if err != nil {
				log.Fatalf("cannot obtain executable path: %s", err)
			}
			log.Infof("Executable path: %s", command)

			if err := eden.StartEServer(cfg.Eden.EServer.Port, cfg.Eden.Images.EServerImageDist, cfg.Eden.EServer.Force, cfg.Eden.EServer.Tag); err != nil {
				log.Errorf("cannot start eserver: %s", err)
			} else {
				log.Infof("Eserver is running and accesible on port %d", cfg.Eden.EServer.Port)
			}
		},
	}

	startEserverCmd.Flags().StringVarP(&cfg.Eden.Images.EServerImageDist, "image-dist", "", "", "image dist for eserver")
	startEserverCmd.Flags().IntVarP(&cfg.Eden.EServer.Port, "eserver-port", "", defaults.DefaultEserverPort, "eserver port")
	startEserverCmd.Flags().StringVarP(&cfg.Eden.EServer.Tag, "eserver-tag", "", defaults.DefaultEServerTag, "tag of eserver container to pull")
	startEserverCmd.Flags().BoolVarP(&cfg.Eden.EServer.Force, "eserver-force", "", false, "eserver force rebuild")

	return startEserverCmd
}

func newStopEserverCmd(cfg *openevec.EdenSetupArgs) *cobra.Command {
	var stopEserverCmd = &cobra.Command{
		Use:   "stop",
		Short: "stop eserver",
		Long:  `Stop eserver.`,
		Run: func(cmd *cobra.Command, args []string) {
			if err := eden.StopEServer(cfg.Runtime.EServerRm); err != nil {
				log.Errorf("cannot stop eserver: %s", err)
			}
		},
	}

	stopEserverCmd.Flags().BoolVarP(&cfg.Runtime.EServerRm, "eserver-rm", "", false, "eserver rm on stop")

	return stopEserverCmd
}

func newStatusEserverCmd(cfg *openevec.EdenSetupArgs) *cobra.Command {
	var statusEserverCmd = &cobra.Command{
		Use:   "status",
		Short: "status of eserver",
		Long:  `Status of eserver.`,
		Run: func(cmd *cobra.Command, args []string) {
			statusEServer, err := eden.StatusEServer()
			if err != nil {
				log.Errorf("cannot obtain status of eserver: %s", err)
			} else {
				fmt.Printf("EServer status: %s\n", statusEServer)
			}
		},
	}
	return statusEserverCmd
}
