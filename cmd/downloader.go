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
	"path/filepath"
	"runtime"
)

var (
	eveArch   string
	eveTag    string
	outputDir string

	newDownload bool
)

var downloaderCmd = &cobra.Command{
	Use:   "download",
	Short: "download eve from docker",
	Long:  `Download eve from docker.`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		assignCobraToViper(cmd)
		if !newDownload {
			if outputDirFlag := cmd.Flags().Lookup("downloader-dist"); outputDirFlag != nil {
				if err := viper.BindPFlag("eve.image-file", outputDirFlag); err != nil {
					log.Fatal(err)
				}
			}
		}
		viperLoaded, err := utils.LoadConfigFile(configFile)
		if err != nil {
			return fmt.Errorf("error reading config: %s", err.Error())
		}
		if viperLoaded {
			eveTag = viper.GetString("eve.tag")
			eveArch = viper.GetString("eve.arch")
			eveHV = viper.GetString("eve.hv")
			adamDist = utils.ResolveAbsPath(viper.GetString("adam.dist"))
			eveImageFile = utils.ResolveAbsPath(viper.GetString("eve.image-file"))
			devModel = viper.GetString("eve.devmodel")
			if newDownload {
				outputDir = filepath.Dir(eveImageFile)
			} else {
				outputDir = eveImageFile
			}
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		if eveTag == "master" {
			eveTag = "latest"
		}
		efiImage := ""
		if newDownload {
			efiImage = fmt.Sprintf("lfedge/eve-uefi:%s-%s", eveTag, eveArch) //download OVMF
			image = fmt.Sprintf("lfedge/eve:%s-%s-%s", eveTag, eveHV, eveArch)
		} else {
			image = fmt.Sprintf("lfedge/eve:%s-%s", eveTag, eveArch) //try download old naming
		}
		log.Debugf("Try ImagePull with (%s)", image)
		if err := utils.PullImage(image); err != nil {
			log.Fatalf("ImagePull (%s): %s", image, err)
		}
		if newDownload {
			configPath := filepath.Join(adamDist, "run", "config")
			if _, err := os.Stat(configPath); os.IsNotExist(err) {
				log.Fatalf("directory not exists: %s", configPath)
			}
			var format string
			var size int
			if devModel == defaults.DefaultRPIModel {
				format = "raw"
				size = 0
			} else {
				format = "qcow2"
				size = defaults.DefaultEVEImageSize
				if err := utils.PullImage(efiImage); err != nil {
					log.Infof("cannot pull %s", efiImage)
					efiImage = fmt.Sprintf("lfedge/eve-uefi") //try with latest version of OVMF
					log.Infof("will retry with %s", efiImage)
					if err := utils.PullImage(efiImage); err != nil {
						log.Fatalf("ImagePull (%s): %s", efiImage, err)
					}
				}
				if err := utils.SaveImage(efiImage, outputDir, ""); err != nil {
					log.Fatalf("SaveImage: %s", err)
				}
			}
			if fileName, err := utils.GenEVEImage(image, outputDir, "live", format, configPath, size); err != nil {
				log.Fatalf("GenEVEImage: %s", err)
			} else {
				if err = utils.CopyFile(fileName, eveImageFile); err != nil {
					log.Fatalf("cannot copy image %s", err)
				}
				if devModel == defaults.DefaultRPIModel {
					log.Infof("Write file %s to sd (it is in raw format)", eveImageFile)
				}
			}
		} else {
			if err := utils.SaveImage(image, outputDir, defaults.DefaultEvePrefixInTar); err != nil {
				log.Fatalf("SaveImage: %s", err)
			}
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
	downloaderCmd.Flags().StringVarP(&eveHV, "eve-hv", "", defaults.DefaultEVEHV, "HV of EVE (kvm or xen)")
	downloaderCmd.Flags().BoolVarP(&newDownload, "new-download", "", defaults.DefaultNewBuildProcess, "use building with docker instead of direct download")
	downloaderCmd.Flags().StringVarP(&outputDir, "downloader-dist", "d", path.Join(currentPath, defaults.DefaultDist, defaults.DefaultEVEDist, "dist", runtime.GOARCH), "output directory")
}
