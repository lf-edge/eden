package cmd

import (
	"github.com/lf-edge/eden/pkg/eden"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"io/ioutil"
	"os"
	"path/filepath"
)

var (
	syslogOutput     string
	eveReleaseOutput string
)

var utilsCmd = &cobra.Command{
	Use:   "utils",
	Short: "Eden utilities",
	Long:  `Additional utilities for EDEN.`,
}

var sdInfoEveCmd = &cobra.Command{
	Use:   "sd <SD_DEVICE_PATH>",
	Short: "get info from SD card",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		eveInfo, err := eden.GetInfoFromSDCard(args[0])
		if err != nil {
			log.Info("Check is EVE on SD and your access to read SD")
			log.Fatalf("Problem with access to EVE partitions: %v", err)
		}
		if eveInfo.EVERelease == nil {
			log.Warning("No eve-release found. Probably, no EVE on SD card")
		} else {
			if err = ioutil.WriteFile(eveReleaseOutput, eveInfo.EVERelease, 0666); err != nil {
				log.Fatal(err)
			} else {
				log.Infof("Your eve-release in %s", eveReleaseOutput)
			}
		}
		if eveInfo.Syslog == nil {
			log.Warning("No syslog found, EVE may not have started yet")
		} else {
			if err = ioutil.WriteFile(syslogOutput, eveInfo.Syslog, 0666); err != nil {
				log.Fatal(err)
			} else {
				log.Infof("Your syslog in %s", syslogOutput)
			}
		}
	},
}

func utilsInit() {
	currentPath, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	utilsCmd.AddCommand(templateCmd)
	utilsCmd.AddCommand(downloaderCmd)
	downloaderInit()
	utilsCmd.AddCommand(ociImageCmd)
	ociImageInit()
	utilsCmd.AddCommand(certsCmd)
	certsInit()
	utilsCmd.AddCommand(gcpCmd)
	gcpInit()
	utilsCmd.AddCommand(sdInfoEveCmd)
	sdInfoEveCmd.Flags().StringVar(&syslogOutput, "syslog-out", filepath.Join(currentPath, "syslog.txt"), "File to save syslog.txt")
	sdInfoEveCmd.Flags().StringVar(&eveReleaseOutput, "everelease-out", filepath.Join(currentPath, "eve-release"), "File to save eve-release")
}
