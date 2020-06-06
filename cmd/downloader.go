package cmd

import (
	"fmt"
	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/utils"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
	"path"
	"runtime"
)

var (
	eveArch   string
	eveTag    string
	outputDir string
)

var downloaderCmd = &cobra.Command{
	Use:   "download",
	Short: "download eve from docker",
	Long:  `Download eve from docker.`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		assingCobraToViper(cmd)
		if outputDirFlag := cmd.Flags().Lookup("downloader-dist"); outputDirFlag != nil {
			if err := viper.BindPFlag("eve.image-file", outputDirFlag); err != nil {
				log.Fatal(err)
			}
		}
		viperLoaded, err := utils.LoadConfigFile(configFile)
		if err != nil {
			return fmt.Errorf("error reading config: %s", err.Error())
		}
		if viperLoaded {
			eveTag = viper.GetString("eve.tag")
			eveArch = viper.GetString("eve.arch")
			outputDir = utils.ResolveAbsPath(viper.GetString("eve.image-file"))
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		if eveTag == "master" {
			eveTag = "latest"
		}
		image = fmt.Sprintf("lfedge/eve:%s-%s", eveTag, eveArch)
		if err := utils.PullImage(image); err != nil {
			log.Fatalf("ImagePull (%s): %s", image, err)
		}
		if err := utils.SaveImage(image, outputDir, defaults.DefaultEvePrefixInTar); err != nil {
			log.Fatalf("SaveImage: %s", err)
		}
	},
}

func downloaderInit() {
	currentPath, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	downloaderCmd.Flags().StringVarP(&eveTag, "eve-tag", "", defaults.DefaultEVETag, "tag to download")
	downloaderCmd.Flags().StringVarP(&eveArch, "eve-arch", "", runtime.GOARCH, "arch of EVE")
	downloaderCmd.Flags().StringVarP(&outputDir, "downloader-dist", "d", path.Join(currentPath, defaults.DefaultDist, defaults.DefaultEVEDist, "dist", runtime.GOARCH), "output directory")
}
