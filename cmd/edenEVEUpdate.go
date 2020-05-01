package cmd

import (
	"fmt"
	"github.com/docker/docker/pkg/fileutils"
	"github.com/lf-edge/eden/pkg/controller"
	"github.com/lf-edge/eden/pkg/controller/einfo"
	"github.com/lf-edge/eden/pkg/utils"
	"github.com/lf-edge/eve/api/go/config"
	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"io/ioutil"
	"os"
	"path"
	"runtime"
)

var (
	eserverIP     string
	baseOSVersion string
	wait          bool
)

var eveUpdateCmd = &cobra.Command{
	Use:   "eve-update <file>",
	Short: "update EVE rootfs",
	Long:  `Update EVE rootfs.`,
	Args:  cobra.MinimumNArgs(1),
	PreRunE: func(cmd *cobra.Command, args []string) error {
		assingCobraToViper(cmd)
		if baseOSVersionFlag := cmd.Flags().Lookup("os-version"); baseOSVersionFlag != nil {
			if err := viper.BindPFlag("eve.base-tag", baseOSVersionFlag); err != nil {
				log.Fatal(err)
			}
		}
		viperLoaded, err := utils.LoadConfigFile(configFile)
		if err != nil {
			return fmt.Errorf("error reading configFile: %s", err.Error())
		}
		if viperLoaded {
			certsIP = viper.GetString("adam.ip")
			adamPort = viper.GetString("adam.port")
			adamDist = utils.ResolveAbsPath(viper.GetString("adam.dist"))
			adamCA = utils.ResolveAbsPath(viper.GetString("adam.ca"))
			eveSSHKey = utils.ResolveAbsPath(viper.GetString("eden.ssh-key"))
			eserverIP = viper.GetString("eden.eserver.ip")
			eserverPort = viper.GetString("eden.eserver.port")
			baseOSVersion = viper.GetString("eve.base-tag")
			eveHV = viper.GetString("eve.hv")
			eveArch = viper.GetString("eve.arch")
			eserverImageDist = utils.ResolveAbsPath(viper.GetString("eden.images.dist"))
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		ctrl, err := controller.CloudPrepare()
		if err != nil {
			log.Fatalf("CloudPrepare: %s", err)
		}
		if err := ctrl.OnBoard(); err != nil {
			log.Fatalf("OnBoard: %s", err)
		}
		dataStore := &config.DatastoreConfig{
			Id:       dataStoreID,
			DType:    config.DsType_DsHttp,
			Fqdn:     fmt.Sprintf("http://%s:%s", eserverIP, eserverPort),
			ApiKey:   "",
			Password: "",
			Dpath:    "",
			Region:   "",
		}
		if _, err := os.Lstat(args[0]); os.IsNotExist(err) {
			log.Fatalf("image file problem: %s", args[0])
		}
		imageFullPath := path.Join(eserverImageDist, "baseos", defaultFilename)
		if _, err := fileutils.CopyFile(args[0], imageFullPath); err != nil {
			log.Fatalf("CopyFile problem: %s", err)
		}
		imageDSPath := fmt.Sprintf("baseos/%s", defaultFilename)
		fi, err := os.Stat(imageFullPath)
		if err != nil {
			log.Fatalf("ImageFile (%s): %s", imageFullPath, err)
		}
		size := fi.Size()

		sha256sum := ""
		sha256sum, err = utils.SHA256SUM(imageFullPath)
		if err != nil {
			log.Fatalf("SHA256SUM (%s): %s", imageFullPath, err)
		}
		img := &config.Image{
			Uuidandversion: &config.UUIDandVersion{
				Uuid:    imageID,
				Version: "4",
			},
			Name:      imageDSPath,
			Sha256:    sha256sum,
			Iformat:   config.Format_QCOW2,
			DsId:      dataStoreID,
			SizeBytes: size,
			Siginfo: &config.SignatureInfo{
				Intercertsurl: "",
				Signercerturl: "",
				Signature:     nil,
			},
		}
		if ds, _ := ctrl.GetDataStore(dataStoreID); ds == nil {
			if err = ctrl.AddDataStore(dataStore); err != nil {
				log.Fatalf("AddDataStore: %s", err)
			}
		}
		if err = ctrl.AddImage(img); err != nil {
			log.Fatalf("AddImage: %s", err)
		}

		if eveHV != "" {
			baseOSVersion = fmt.Sprintf("%s-%s-%s", baseOSVersion, eveHV, eveArch)
		} else {
			baseOSVersion = fmt.Sprintf("%s-%s", baseOSVersion, eveArch)
		}
		if err = ctrl.AddBaseOsConfig(&config.BaseOSConfig{
			Uuidandversion: &config.UUIDandVersion{
				Uuid:    baseID,
				Version: "4",
			},
			Drives: []*config.Drive{{
				Image:        img,
				Readonly:     false,
				Preserve:     false,
				Drvtype:      config.DriveType_Unclassified,
				Target:       config.Target_TgtUnknown,
				Maxsizebytes: img.SizeBytes,
			}},
			Activate:      true,
			BaseOSVersion: baseOSVersion,
			BaseOSDetails: nil,
		}); err != nil {
			log.Fatalf("AddBaseOsConfig: %s", err)
		}

		devices, err := ctrl.DeviceList()
		if err != nil {
			log.Fatalf("DeviceList: %s", err)
		}
		for _, devID := range devices {
			devUUID, err := uuid.FromString(devID)
			if err != nil {
				log.Fatalf("uuidGet: %s", err)
			}

			deviceCtx, err := ctrl.GetDeviceUUID(devUUID)
			if err != nil {
				deviceCtx, err = ctrl.AddDevice(devUUID)
				{
					log.Fatal("Fail in add device: ", err)
				}
			}
			if eveSSHKey != "" {
				b, err := ioutil.ReadFile(eveSSHKey)
				switch {
				case err != nil && os.IsNotExist(err):
					log.Fatalf("sshKey file %s does not exist", eveSSHKey)
				case err != nil:
					log.Fatalf("error reading sshKey file %s: %v", eveSSHKey, err)
				}
				deviceCtx.SetSSHKeys([]string{string(b)})
			}
			deviceCtx.SetBaseOSConfig([]string{baseID})
			err = ctrl.ConfigSync(deviceCtx)
			log.Info("Request for update sended")
			if wait {
				log.Info("Please wait for operation ending")
				if err := ctrl.InfoChecker(devUUID, map[string]string{"devId": devUUID.String(), "shortVersion": baseOSVersion}, einfo.ZInfoDevSW, einfo.HandleFirst, einfo.InfoAny, 500); err != nil {
					log.Fatal("Fail in waiting for base image update init: ", err)
				}
				log.Info("Request for update received by EVE")
				if err := ctrl.InfoChecker(devUUID, map[string]string{"devId": devUUID.String(), "shortVersion": baseOSVersion, "downloadProgress": "100"}, einfo.ZInfoDevSW, einfo.HandleFirst, einfo.InfoAny, 1000); err != nil {
					log.Fatal("Fail in waiting for base image download progress: ", err)
				}
				log.Info("New image downloaded by EVE")
				if err := ctrl.InfoChecker(devUUID, map[string]string{"devId": devUUID.String(), "shortVersion": baseOSVersion, "status": "INSTALLED", "partitionState": "(inprogress|active)"}, einfo.ZInfoDevSW, einfo.HandleFirst, einfo.InfoAny, 1000); err != nil {
					log.Fatal("Fail in waiting for base image installed status: ", err)
				}
				log.Info("Update done")
			}
			break
		}

	},
}

func eveUpdateInit() {
	eveUpdateCmd.Flags().StringVar(&configFile, "configFile", "", "path to configFile file")
	eveUpdateCmd.Flags().StringVar(&eserverIP, "eserver-ip", "", "IP of eserver for EVE access")
	eveUpdateCmd.Flags().StringVarP(&eserverPort, "eserver-port", "", "8888", "eserver port")
	eveUpdateCmd.Flags().StringVarP(&baseOSVersion, "os-version", "", utils.DefaultBaseOSTag, "version of ROOTFS")
	eveUpdateCmd.Flags().StringVarP(&eveHV, "hv", "", "kvm", "hv of rootfs to use")
	eveUpdateCmd.Flags().StringVarP(&eveArch, "eve-arch", "", runtime.GOARCH, "EVE arch")
	eveUpdateCmd.Flags().BoolVar(&wait, "wait", true, "wait for system update")
}
