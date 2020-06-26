package cmd

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/utils"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/crypto/pbkdf2"
	"golang.org/x/crypto/ssh/terminal"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

var (
	eveDist   string
	eveRepo   string
	download  bool
	binDir    string
	dockerYML string
	vmYML     string
	force     bool
	dryRun    bool
	apiV1     bool

	devModel string
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "setup harness",
	Long:  `Setup harness.`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		assingCobraToViper(cmd)
		viperLoaded, err := utils.LoadConfigFile(configFile)
		if err != nil {
			return fmt.Errorf("error reading config: %s", err.Error())
		}
		if viperLoaded {
			download = viper.GetBool("eden.download")
			binDir = utils.ResolveAbsPath(viper.GetString("eden.bin-dist"))
			dockerYML = utils.ResolveAbsPath(viper.GetString("eden.images.docker"))
			vmYML = utils.ResolveAbsPath(viper.GetString("eden.images.vm"))
			//certs
			certsDir = utils.ResolveAbsPath(viper.GetString("eden.certs-dist"))
			//adam
			adamTag = viper.GetString("adam.tag")
			adamPort = viper.GetInt("adam.port")
			adamDist = utils.ResolveAbsPath(viper.GetString("adam.dist"))
			certsDomain = viper.GetString("adam.domain")
			certsIP = viper.GetString("adam.ip")
			certsEVEIP = viper.GetString("adam.eve-ip")
			apiV1 = viper.GetBool("adam.v1")
			//eve
			qemuFirmware = viper.GetStringSlice("eve.firmware")
			qemuConfigPath = utils.ResolveAbsPath(viper.GetString("eve.config-part"))
			qemuDTBPath = utils.ResolveAbsPath(viper.GetString("eve.dtb-part"))
			eveImageFile = utils.ResolveAbsPath(viper.GetString("eve.image-file"))
			certsUUID = viper.GetString("eve.uuid")
			eveDist = utils.ResolveAbsPath(viper.GetString("eve.dist"))
			eveRepo = viper.GetString("eve.repo")
			eveTag = viper.GetString("eve.tag")
			eveHV = viper.GetString("eve.hv")
			eveArch = viper.GetString("eve.arch")
			qemuHostFwd = viper.GetStringMapString("eve.hostfwd")
			qemuFileToSave = utils.ResolveAbsPath(viper.GetString("eve.qemu-config"))
			//eserver
			eserverImageDist = utils.ResolveAbsPath(viper.GetString("eden.images.dist"))

			devModel = viper.GetString("eve.devmodel")

			ssid = viper.GetString("eve.ssid")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		command, err := os.Executable()
		if err != nil {
			log.Fatalf("cannot obtain executable path: %s", err)
		}
		if _, err := os.Stat(filepath.Join(certsDir, "server.pem")); os.IsNotExist(err) {
			wifiPSK := ""
			if ssid != "" {
				fmt.Printf("Enter password for wifi %s: ", ssid)
				pass, _ := terminal.ReadPassword(0)
				wifiPSK = strings.ToLower(hex.EncodeToString(pbkdf2.Key(pass, []byte(ssid), 4096, 32, sha1.New)))
				fmt.Println()
			}
			if err := utils.GenerateEveCerts(command, certsDir, certsDomain, certsIP, certsEVEIP, certsUUID, ssid, wifiPSK); err != nil {
				log.Errorf("cannot GenerateEveCerts: %s", err)
			} else {
				log.Info("GenerateEveCerts done")
			}
		} else {
			log.Infof("Certs already exists in certs dir: %s", certsDir)
		}
		if _, err := os.Stat(filepath.Join(adamDist, "run", "config", "server.pem")); os.IsNotExist(err) {
			if err := utils.CopyCertsToAdamConfig(certsDir, certsDomain, certsEVEIP, adamPort, adamDist, apiV1); err != nil {
				log.Errorf("cannot CopyCertsToAdamConfig: %s", err)
			} else {
				log.Info("CopyCertsToAdamConfig done")
			}
		} else {
			log.Infof("Certs already exists in adam dir: %s", certsDir)
		}
		var imageFormat string
		switch devModel {
		case defaults.DefaultRPIModel:
			// don't second guess explicit rpi- setting
			if !strings.HasPrefix(eveHV, "rpi-") {
				if eveHV == "kvm" {
					eveHV = fmt.Sprintf("rpi-%s", eveHV)
				} else {
					eveHV = "rpi"
				}
			}
			imageFormat = "raw"
		case defaults.DefaultEVEModel:
			imageFormat = "qcow2"
		default:
			log.Fatalf("Unsupported dev model %s", devModel)
		}
		if !download {
			if _, err := os.Lstat(eveImageFile); os.IsNotExist(err) {
				if err := utils.CloneFromGit(eveDist, eveRepo, eveTag); err != nil {
					log.Errorf("cannot clone EVE: %s", err)
				} else {
					log.Info("clone EVE done")
				}
				builedImage := ""
				builedAdditional := ""
				if builedImage, builedAdditional, err = utils.MakeEveInRepo(eveDist, adamDist, eveArch, eveHV, imageFormat, false); err != nil {
					log.Errorf("cannot MakeEveInRepo: %s", err)
				} else {
					log.Info("MakeEveInRepo done")
				}
				if err = utils.CopyFile(builedImage, eveImageFile); err != nil {
					log.Fatal(err)
				}
				if builedAdditional != "" {
					if err = utils.CopyFile(builedAdditional, filepath.Join(filepath.Dir(eveImageFile), filepath.Base(builedAdditional))); err != nil {
						log.Fatal(err)
					}
				}
				if devModel == defaults.DefaultRPIModel {
					log.Infof("Write file %s to sd (it is in raw format)", eveImageFile)
				}
			} else {
				log.Infof("EVE already exists in dir: %s", eveDist)

			}
		} else {
			if _, err := os.Lstat(eveImageFile); os.IsNotExist(err) {
				if err := utils.DownloadEveFormDocker(command, eveDist, eveArch, eveHV, eveTag, defaults.DefaultNewBuildProcess); err != nil {
					log.Errorf("cannot download EVE: %s", err)
				} else {
					log.Info("download EVE done")
				}
				if !defaults.DefaultNewBuildProcess {
					if err := utils.ChangeConfigPartAndRootFs(command, eveDist, adamDist, eveArch, eveHV); err != nil {
						log.Errorf("cannot ChangeConfigPartAndRootFs EVE: %s", err)
					} else {
						log.Info("ChangeConfigPartAndRootFs EVE done")
					}
				}
			} else {
				log.Infof("EVE already exists in dir: %s", eveDist)
			}
		}
	},
}

func setupInit() {
	currentPath, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	setupCmd.Flags().StringVarP(&certsDir, "certs-dist", "o", filepath.Join(currentPath, defaults.DefaultDist, defaults.DefaultCertsDist), "directory with certs")
	setupCmd.Flags().StringVarP(&certsDomain, "domain", "d", defaults.DefaultDomain, "FQDN for certificates")
	setupCmd.Flags().StringVarP(&certsIP, "ip", "i", defaults.DefaultIP, "IP address to use")
	setupCmd.Flags().StringVarP(&certsEVEIP, "eve-ip", "", defaults.DefaultEVEIP, "IP address to use for EVE")
	setupCmd.Flags().StringVarP(&certsUUID, "uuid", "u", defaults.DefaultUUID, "UUID to use for device")

	setupCmd.Flags().StringVarP(&adamTag, "adam-tag", "", defaults.DefaultAdamTag, "Adam tag")
	setupCmd.Flags().StringVarP(&adamDist, "adam-dist", "", filepath.Join(currentPath, defaults.DefaultDist, defaults.DefaultAdamDist), "adam dist to start (required)")
	setupCmd.Flags().IntVarP(&adamPort, "adam-port", "", defaults.DefaultAdamPort, "adam dist to start")

	setupCmd.Flags().StringSliceVarP(&qemuFirmware, "eve-firmware", "", nil, "firmware path")
	setupCmd.Flags().StringVarP(&qemuConfigPath, "config-part", "", "", "path for config drive")
	setupCmd.Flags().StringVarP(&qemuDTBPath, "dtb-part", "", "", "path for device tree drive (for arm)")
	setupCmd.Flags().StringVarP(&eveImageFile, "image-file", "", "", "path for image drive (required)")
	setupCmd.Flags().StringVarP(&eveDist, "eve-dist", "", filepath.Join(currentPath, defaults.DefaultDist, defaults.DefaultEVEDist), "directory to save EVE")
	setupCmd.Flags().StringVarP(&eveRepo, "eve-repo", "", defaults.DefaultEveRepo, "EVE repo")
	setupCmd.Flags().StringVarP(&eveTag, "eve-tag", "", defaults.DefaultEVETag, "EVE tag")
	setupCmd.Flags().StringVarP(&eveArch, "eve-arch", "", runtime.GOARCH, "EVE arch")
	setupCmd.Flags().StringToStringVarP(&qemuHostFwd, "eve-hostfwd", "", defaults.DefaultQemuHostFwd, "port forward map")
	setupCmd.Flags().StringVarP(&qemuFileToSave, "qemu-config", "", filepath.Join(currentPath, defaults.DefaultDist, defaults.DefaultQemuFileToSave), "file to save qemu config")
	setupCmd.Flags().BoolVarP(&download, "download", "", true, "download EVE or build")
	setupCmd.Flags().StringVarP(&eveHV, "eve-hv", "", defaults.DefaultEVEHV, "hv of rootfs to use")

	setupCmd.Flags().StringVarP(&eserverImageDist, "image-dist", "", filepath.Join(currentPath, defaults.DefaultDist, defaults.DefaultImageDist), "image dist for eserver")
	setupCmd.Flags().StringVarP(&binDir, "bin-dist", "", filepath.Join(currentPath, defaults.DefaultDist, defaults.DefaultBinDist), "directory for binaries")
	setupCmd.Flags().BoolVarP(&force, "force", "", false, "force overwrite config file")
	setupCmd.Flags().BoolVarP(&dryRun, "dry-run", "", false, "")
	setupCmd.Flags().BoolVarP(&apiV1, "api-v1", "", true, "use v1 api")
}
