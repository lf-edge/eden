package cmd

import (
	"fmt"
	"path/filepath"
	"runtime"

	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/models"
	"github.com/lf-edge/eden/pkg/utils"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	eveArch    string
	eveTag     string
	eveUefiTag string
	outputDir  string
)

var downloaderCmd = &cobra.Command{
	Use: "download",
}
var downloadEVECmd = &cobra.Command{
	Use:   "eve",
	Short: "download eve live image from docker",
	Long:  `Download eve live image from docker.`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		assignCobraToViper(cmd)
		viperLoaded, err := utils.LoadConfigFile(configFile)
		if err != nil {
			return fmt.Errorf("error reading config: %s", err.Error())
		}
		if viperLoaded {
			eveRegistry = viper.GetString("eve.registry")
			eveTag = viper.GetString("eve.tag")
			eveUefiTag = viper.GetString("eve.uefi-tag")
			eveArch = viper.GetString("eve.arch")
			eveHV = viper.GetString("eve.hv")
			adamDist = utils.ResolveAbsPath(viper.GetString("adam.dist"))
			eveImageFile = utils.ResolveAbsPath(viper.GetString("eve.image-file"))
			devModel = viper.GetString("eve.devmodel")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		model, err := models.GetDevModelByName(devModel)
		if err != nil {
			log.Fatalf("GetDevModelByName: %s", err)
		}
		format := model.DiskFormat()
		eveDesc := utils.EVEDescription{
			ConfigPath:  adamDist,
			Arch:        eveArch,
			HV:          eveHV,
			Registry:    eveRegistry,
			Tag:         eveTag,
			Format:      format,
			ImageSizeMB: eveImageSizeMB,
		}
		if err := utils.DownloadEveLive(eveDesc, eveImageFile); err != nil {
			log.Fatal(err)
		}
		if format == "qcow2" {
			uefiDesc := utils.UEFIDescription{
				Registry: eveRegistry,
				Tag:      eveUefiTag,
				Arch:     eveArch,
			}
			if err := utils.DownloadUEFI(uefiDesc, filepath.Dir(eveImageFile)); err != nil {
				log.Fatal(err)
			}
		}
		log.Infof(model.DiskReadyMessage(), eveImageFile)
		fmt.Println(eveImageFile)
	},
}
var downloadEVERootFSCmd = &cobra.Command{
	Use:   "eve-rootfs",
	Short: "download eve rootfs image from docker",
	Long:  `Download eve rootfs image from docker.`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		assignCobraToViper(cmd)
		viperLoaded, err := utils.LoadConfigFile(configFile)
		if err != nil {
			return fmt.Errorf("error reading config: %s", err.Error())
		}
		if viperLoaded {
			eveTag = viper.GetString("eve.tag")
			eveArch = viper.GetString("eve.arch")
			eveHV = viper.GetString("eve.hv")
			eveRegistry = viper.GetString("eve.registry")
			if outputDir == "" {
				outputDir = filepath.Dir(utils.ResolveAbsPath(viper.GetString("eve.image-file")))
			}
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		eveDesc := utils.EVEDescription{
			ConfigPath:  certsDir,
			Arch:        eveArch,
			HV:          eveHV,
			Registry:    eveRegistry,
			Tag:         eveTag,
			Format:      imageFormat,
			ImageSizeMB: eveImageSizeMB,
		}
		image, err := utils.DownloadEveRootFS(eveDesc, outputDir)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(image)
	},
}

func downloaderInit() {
	downloaderCmd.AddCommand(downloadEVECmd)
	downloadEVECmd.Flags().StringVarP(&eveTag, "eve-tag", "", defaults.DefaultEVETag, "tag to download eve")
	downloadEVECmd.Flags().StringVarP(&eveUefiTag, "eve-uefi-tag", "", defaults.DefaultEVETag, "tag to download eve UEFI")
	downloadEVECmd.Flags().StringVarP(&eveArch, "eve-arch", "", runtime.GOARCH, "arch of EVE")
	downloadEVECmd.Flags().StringVarP(&eveHV, "eve-hv", "", defaults.DefaultEVEHV, "HV of EVE (kvm or xen)")
	downloadEVECmd.Flags().StringVarP(&eveImageFile, "image-file", "i", "", "path for image drive")
	downloadEVECmd.Flags().StringVarP(&adamDist, "adam-dist", "", "", "adam dist to start")
	downloadEVECmd.Flags().IntVar(&eveImageSizeMB, "image-size", defaults.DefaultEVEImageSize, "Image size of EVE in MB")
	downloaderCmd.AddCommand(downloadEVERootFSCmd)
	downloadEVERootFSCmd.Flags().StringVarP(&eveTag, "eve-tag", "", defaults.DefaultEVETag, "tag to download")
	downloadEVERootFSCmd.Flags().StringVarP(&eveArch, "eve-arch", "", runtime.GOARCH, "arch of EVE")
	downloadEVERootFSCmd.Flags().StringVarP(&eveHV, "eve-hv", "", defaults.DefaultEVEHV, "HV of EVE (kvm or xen)")
	downloadEVERootFSCmd.Flags().StringVarP(&outputDir, "downloader-dist", "d", "", "output directory")
}
