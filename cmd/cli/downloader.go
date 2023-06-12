package cmd

import (
	"fmt"
	"path/filepath"
	"runtime"

	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/models"
	"github.com/lf-edge/eden/pkg/openevec"
	"github.com/lf-edge/eden/pkg/utils"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func newDownloaderCmd(cfg *openevec.EdenSetupArgs) *cobra.Command {
	var downloaderCmd = &cobra.Command{
		Use: "download",
	}

	groups := CommandGroups{
		{
			Message: "Basic Commands",
			Commands: []*cobra.Command{
				newDownloadEVECmd(cfg),
				newDownloadEVERootFSCmd(cfg),
			},
		},
	}

	groups.AddTo(downloaderCmd)

	return downloaderCmd
}

func newDownloadEVERootFSCmd(cfg *openevec.EdenSetupArgs) *cobra.Command {
	outputDir := ""

	var downloadEVERootFSCmd = &cobra.Command{
		Use:   "eve-rootfs",
		Short: "download eve rootfs image from docker",
		Long:  `Download eve rootfs image from docker.`,
		Run: func(cmd *cobra.Command, args []string) {
			if outputDir == "" {
				outputDir = filepath.Dir(cfg.Eve.ImageFile)
			}
			model, err := models.GetDevModelByName(cfg.Eve.DevModel)
			if err != nil {
				log.Fatalf("GetDevModelByName: %s", err)
			}
			format := model.DiskFormat()
			eveDesc := utils.EVEDescription{
				ConfigPath:  cfg.Adam.Dist,
				Arch:        cfg.Eve.Arch,
				HV:          cfg.Eve.HV,
				Registry:    cfg.Eve.Registry,
				Tag:         cfg.Eve.Tag,
				Format:      format,
				ImageSizeMB: cfg.Eve.ImageSizeMB,
			}
			image, err := utils.DownloadEveRootFS(eveDesc, outputDir)
			if err != nil {
				log.Fatal(err)
			}
			fmt.Println(image)
		},
	}

	downloadEVERootFSCmd.Flags().StringVarP(&cfg.Eve.Tag, "eve-tag", "", defaults.DefaultEVETag, "tag to download")
	downloadEVERootFSCmd.Flags().StringVarP(&cfg.Eve.Arch, "eve-arch", "", runtime.GOARCH, "arch of EVE")
	downloadEVERootFSCmd.Flags().StringVarP(&cfg.Eve.HV, "eve-hv", "", defaults.DefaultEVEHV, "HV of EVE (kvm or xen)")
	downloadEVERootFSCmd.Flags().StringVarP(&outputDir, "downloader-dist", "d", "", "output directory")

	return downloadEVERootFSCmd
}

func newDownloadEVECmd(cfg *openevec.EdenSetupArgs) *cobra.Command {
	var downloadEVECmd = &cobra.Command{
		Use:   "eve",
		Short: "download eve live image from docker",
		Long:  `Download eve live image from docker.`,
		Run: func(cmd *cobra.Command, args []string) {
			if err := openevec.DownloadEve(cfg); err != nil {
				log.Fatal(err)
			}
		},
	}

	downloadEVECmd.Flags().StringVarP(&cfg.Eve.Tag, "eve-tag", "", defaults.DefaultEVETag, "tag to download eve")
	downloadEVECmd.Flags().StringVarP(&cfg.Eve.UefiTag, "eve-uefi-tag", "", defaults.DefaultEVETag, "tag to download eve UEFI")
	downloadEVECmd.Flags().StringVarP(&cfg.Eve.Arch, "eve-arch", "", runtime.GOARCH, "arch of EVE")
	downloadEVECmd.Flags().StringVarP(&cfg.Eve.HV, "eve-hv", "", defaults.DefaultEVEHV, "HV of EVE (kvm or xen)")
	downloadEVECmd.Flags().StringVarP(&cfg.Eve.ImageFile, "image-file", "i", "", "path for image drive")
	downloadEVECmd.Flags().StringVarP(&cfg.Adam.Dist, "adam-dist", "", cfg.Adam.Dist, "adam dist to start")
	downloadEVECmd.Flags().IntVar(&cfg.Eve.ImageSizeMB, "image-size", defaults.DefaultEVEImageSize, "Image size of EVE in MB")

	return downloadEVECmd
}
