package cmd

import (
	"fmt"
	"github.com/lf-edge/eden/pkg/utils"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
	"path"
	"runtime"
)

const (
	defaultEveTag         = "5.1.11"
	defaultEvePrefixInTar = "bits"
)

var (
	eveArch   string
	eveTag    string
	outputDir string
	saveLocal bool
	baseos    bool
)

var downloaderCmd = &cobra.Command{
	Use:   "download",
	Short: "download eve from docker",
	Long:  `Download eve from docker.`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		viperLoaded, err := utils.LoadConfigFile(config)
		if err != nil {
			return fmt.Errorf("error reading config: %s", err.Error())
		}
		if viperLoaded {
			if !baseos {
				eveTag = viper.GetString("eve-tag")
			} else {
				eveTag = viper.GetString("eve-base-tag")
			}
			eveArch = viper.GetString("eve-arch")
			outputDir = viper.GetString("downloader-dist")
			saveLocal = viper.GetBool("downloader-save")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		if eveTag == "master" {
			eveTag = "latest"
		}
		image = fmt.Sprintf("lfedge/eve:%s-%s", eveTag, eveArch)
		if err := utils.PullImage(image); err != nil {
			log.Fatalf("ImagePull: %s", err)
		}
		if err := utils.SaveImage(image, outputDir, defaultEvePrefixInTar); err != nil {
			log.Fatalf("SaveImage: %s", err)
		}
	},
}

func downloaderInit() {
	currentPath, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	downloaderCmd.Flags().StringVarP(&eveTag, "eve-tag", "", defaultEveTag, "tag to download")
	downloaderCmd.Flags().StringVarP(&eveArch, "eve-arch", "", runtime.GOARCH, "arch of EVE")
	downloaderCmd.Flags().StringVarP(&outputDir, "downloader-dist", "d", path.Join(currentPath, "dist", "eve", "dist", runtime.GOARCH), "output directory")
	downloaderCmd.Flags().BoolVarP(&saveLocal, "downloader-save", "", true, "save image to local docker registry")
	downloaderCmd.Flags().BoolVarP(&baseos, "baseos", "", false, "base OS download")
	if err := viper.BindPFlags(downloaderCmd.Flags()); err != nil {
		log.Fatal(err)
	}
	downloaderCmd.Flags().StringVar(&config, "config", "", "path to config file")
}
