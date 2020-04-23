package cmd

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/diskfs/go-diskfs"
	"github.com/diskfs/go-diskfs/disk"
	"github.com/diskfs/go-diskfs/filesystem"
	"github.com/lf-edge/eden/pkg/utils"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

var eveImageFile = ""
var eveHV = ""

var confChangerCmd = &cobra.Command{
	Use:   "confchanger",
	Short: "change config in EVE image",
	Long:  `Change config in EVE image.`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		viperLoaded, err := utils.LoadViperConfig(config)
		if err != nil {
			return fmt.Errorf("error reading config: %s", err.Error())
		}
		if viperLoaded {
			eveImageFile = viper.GetString("image-file")
			qemuConfigPath = viper.GetString("config-part")
			eveHV = viper.GetString("hv")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		eveImageFilePath, err := filepath.Abs(eveImageFile)
		if err != nil {
			log.Fatalf("image-file problems: %s", err)
		}
		qemuConfigPathAbs, err := filepath.Abs(qemuConfigPath)
		if err != nil {
			log.Fatalf("config-part problems: %s", err)
		}
		filename := filepath.Base(eveImageFilePath)
		tempFilePath := filepath.Join(filepath.Dir(eveImageFilePath), fmt.Sprintf("%s.raw", filename))
		_, stderr, err := utils.RunCommandAndWait("qemu-img", strings.Fields(fmt.Sprintf("convert -O raw %s %s", eveImageFilePath, tempFilePath))...)
		defer os.Remove(tempFilePath)
		if err != nil {
			log.Error(stderr)
			log.Fatal(err)
		}
		diskOpen, err := diskfs.Open(tempFilePath)
		if err != nil {
			log.Fatal(err)
		}
		pt, err := diskOpen.GetPartitionTable()
		if err != nil {
			log.Fatal(err)
		}
		diskOpen.Table = pt
		rootFSSize, err := pt.GetPartitionSize(2)
		if err != nil {
			log.Fatal(err)
		}
		rootFSPath := filepath.Join(filepath.Dir(eveImageFilePath), "installer", fmt.Sprintf("rootfs-%s.img", eveHV))
		info, err := os.Lstat(rootFSPath)
		if err != nil {
			log.Fatal(err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			//follow symlinks
			rootFSPath, err = os.Readlink(rootFSPath)
			if err != nil {
				log.Fatalf("EvalSymlinks: %s", err)
			}
		}
		//use rootfs with selected HV
		file, err := os.Open(filepath.Join(filepath.Dir(eveImageFilePath), "installer", filepath.Base(rootFSPath)))
		if err != nil {
			log.Fatalf("diskRootFS: %s", err)
		}
		fileStat, err := file.Stat()
		if err != nil {
			log.Fatalf("diskRootFS Stat: %s", err)
		}
		//fill bytes with empty values
		buf := make([]byte, rootFSSize-fileStat.Size())
		joinedFile := io.MultiReader(file, bytes.NewReader(buf))
		pr := bufio.NewReader(joinedFile)
		//copy to partition
		if _, err = diskOpen.WritePartitionContents(2, pr); err != nil {
			log.Fatalf("WritePartitionContents: %s", err)
		}
		if err = file.Close(); err != nil {
			log.Fatal(err)
		}
		fSpec := disk.FilesystemSpec{Partition: 4, FSType: filesystem.TypeFat32, VolumeLabel: "EVE"}
		fs, err := diskOpen.CreateFilesystem(fSpec)
		if err != nil {
			log.Fatal(err)
		}
		err = filepath.Walk(qemuConfigPathAbs,
			func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if info.IsDir() {
					err = fs.Mkdir(filepath.Join("/", strings.TrimPrefix(path, qemuConfigPathAbs)))
					if err != nil {
						log.Fatal(err)
					}
					return nil
				}
				rw, err := fs.OpenFile(filepath.Join("/", strings.TrimPrefix(path, qemuConfigPathAbs)), os.O_CREATE|os.O_RDWR)
				if err != nil {
					log.Fatal(err)
				}
				content, err := ioutil.ReadFile(path)
				if err != nil {
					log.Fatal(err)
				}
				_, err = rw.Write(content)
				if err != nil {
					log.Fatal(err)
				}
				return nil
			})
		if err != nil {
			log.Fatal(err)
		}
		err = os.Remove(eveImageFilePath)
		if err != nil {
			log.Fatal(err)
		}
		_, stderr, err = utils.RunCommandAndWait("qemu-img", strings.Fields(fmt.Sprintf("convert -c -O qcow2 %s %s", tempFilePath, eveImageFilePath))...)
		if err != nil {
			log.Error(stderr)
			log.Fatal(err)
		}
	},
}

func confChangerInit() {
	currentPath, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	confChangerCmd.Flags().StringVarP(&eveImageFile, "image-file", "", filepath.Join(currentPath, "dist", "eve", "dist", runtime.GOARCH, "live.qcow2"), "image to modify (required)")
	confChangerCmd.Flags().StringVarP(&qemuConfigPath, "config-part", "", filepath.Join(currentPath, "dist", "adam", "run", "config"), "path for config drive")
	confChangerCmd.Flags().StringVarP(&eveHV, "hv", "", "kvm", "hv of rootfs to use")
	if err := viper.BindPFlags(confChangerCmd.Flags()); err != nil {
		log.Fatal(err)
	}
	confChangerCmd.Flags().StringVar(&config, "config", "", "path to config file")
}
