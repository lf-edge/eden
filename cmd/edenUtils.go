package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/eden"
	"github.com/lf-edge/eden/pkg/utils"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
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
			}
			log.Infof("Your eve-release in %s", eveReleaseOutput)
		}
		if eveInfo.Syslog == nil {
			log.Warning("No syslog found, EVE may not have started yet")
		} else {
			if err = ioutil.WriteFile(syslogOutput, eveInfo.Syslog, 0666); err != nil {
				log.Fatal(err)
			}
			log.Infof("Your syslog in %s", syslogOutput)
		}
	},
}

var uploadGitCmd = &cobra.Command{
	Use: "gitupload <file or directory> " +
		"<git repo in notation https://GIT_LOGIN:GIT_TOKEN@GIT_REPO> <branch> [directory in git]",
	Long: "Upload file or directory to provided git branch into directory with the same name as branch " +
		"or into provided directory",
	Args: cobra.RangeArgs(3, 4),
	Run: func(cmd *cobra.Command, args []string) {
		if _, err := os.Stat(args[0]); os.IsNotExist(err) {
			log.Fatal(err)
		}
		absPath, err := filepath.Abs(args[0])
		if err != nil {
			log.Fatal(err)
		}
		image := fmt.Sprintf("%s:%s", defaults.DefaultProcContainerRef, defaults.DefaultProcTag)
		directoryToSaveOnGit := args[2]
		if len(args) == 4 {
			directoryToSaveOnGit = args[3]
		}
		commandToRun := fmt.Sprintf("-i /in/%s -o %s -b %s -d %s git",
			filepath.Base(absPath), args[1], args[2], directoryToSaveOnGit)
		volumeMap := map[string]string{"/in": filepath.Dir(absPath)}
		var result string
		if result, err = utils.RunDockerCommand(image, commandToRun, volumeMap); err != nil {
			log.Fatal(err)
		}
		fmt.Println(result)
	},
}

func utilsInit() {
	currentPath, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	utilsCmd.AddCommand(completionCmd)
	utilsCmd.AddCommand(templateCmd)
	utilsCmd.AddCommand(downloaderCmd)
	downloaderInit()
	utilsCmd.AddCommand(ociImageCmd)
	ociImageInit()
	utilsCmd.AddCommand(certsCmd)
	certsInit()
	utilsCmd.AddCommand(gcpCmd)
	gcpInit()
	utilsCmd.AddCommand(asbddsCmd)
	asbddsInit()
	utilsCmd.AddCommand(sdInfoEveCmd)
	debugInit()
	utilsCmd.AddCommand(debugCmd)
	utilsCmd.AddCommand(uploadGitCmd)
	sdInfoEveCmd.Flags().StringVar(&syslogOutput, "syslog-out", filepath.Join(currentPath, "syslog.txt"), "File to save syslog.txt")
	sdInfoEveCmd.Flags().StringVar(&eveReleaseOutput, "everelease-out", filepath.Join(currentPath, "eve-release"), "File to save eve-release")
}
