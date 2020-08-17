package cmd

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"

	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/utils"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/crypto/pbkdf2"
	"golang.org/x/crypto/ssh/terminal"
)

var (
	eveDist  string
	eveRepo  string
	download bool
	binDir   string
	eveHV    string
	force    bool
	dryRun   bool
	apiV1    bool

	devModel string
)

func configCheck() {
	sconf := utils.ResolveAbsPath(defaults.DefaultConfigSaved)

	abs, err := filepath.Abs(sconf)
	if err != nil {
		log.Fatalf("fail in reading filepath: %s\n", err.Error())
		os.Exit(-2)
	}

	if _, err = os.Lstat(abs); os.IsNotExist(err) {
		if err = utils.CopyFile(configFile, abs); err != nil {
			log.Fatalf("copying fail %s\n", err.Error())
			os.Exit(-3)
		}
	} else {

		viperLoaded, err := utils.LoadConfigFile(abs)
		if err != nil {
			log.Fatalf("error reading config %s: %s\n", abs, err.Error())
			os.Exit(-2)
		}
		if viperLoaded {
			confOld := viper.AllSettings()

			viperLoaded, err = utils.LoadConfigFile(configFile)
			if err != nil {
				log.Fatalf("error reading config %s: %s", configFile, err.Error())
				os.Exit(-2)
			}

			confCur := viper.AllSettings()

			if reflect.DeepEqual(confOld, confCur) {
				log.Infof("Config file %s is the same as %s\n", configFile, sconf)
			} else {
				log.Fatalf("The current configuration file %s is different from the saved %s. You can fix this with the commands 'eden config clean' and 'eden config add/set/edit'.\n", configFile, abs)
				os.Exit(-1)
			}
		} else {
			/* Incorrect saved config -- just rewrite by current */
			if err = utils.CopyFile(configFile, abs); err != nil {
				log.Fatalf("copying fail %s\n", err.Error())
				os.Exit(-3)
			}
		}
	}
}

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "setup harness",
	Long:  `Setup harness.`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		assignCobraToViper(cmd)

		viperLoaded, err := utils.LoadConfigFile(configFile)
		if err != nil {
			return fmt.Errorf("error reading config: %s", err.Error())
		}

		configCheck()

		if viperLoaded {
			download = viper.GetBool("eden.download")
			binDir = utils.ResolveAbsPath(viper.GetString("eden.bin-dist"))
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
			eveUefiTag = viper.GetString("eve.uefi-tag")
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
		if _, err := os.Stat(filepath.Join(certsDir, "root-certificate.pem")); os.IsNotExist(err) {
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
		if err := utils.GenerateEVEConfig(certsDir, certsDomain, certsEVEIP, adamPort, apiV1); err != nil {
			log.Errorf("cannot GenerateEVEConfig: %s", err)
		} else {
			log.Info("GenerateEVEConfig done")
		}
		var imageFormat string
		switch devModel {
		case defaults.DefaultRPIModel:
			// don't second guess explicit rpi- setting
			if !strings.HasPrefix(eveHV, "rpi-") {
				eveHV = fmt.Sprintf("rpi-%s", eveHV)
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
				if builedImage, builedAdditional, err = utils.MakeEveInRepo(eveDist, certsDir, eveArch, eveHV, imageFormat, false); err != nil {
					log.Errorf("cannot MakeEveInRepo: %s", err)
				} else {
					log.Info("MakeEveInRepo done")
				}
				if err = utils.CopyFile(builedImage, eveImageFile); err != nil {
					log.Fatal(err)
				}
				builedAdditionalSplitted := strings.Split(builedAdditional, ",")
				for _, additionalFile := range builedAdditionalSplitted {
					if additionalFile != "" {
						if err = utils.CopyFile(additionalFile, filepath.Join(filepath.Dir(eveImageFile), filepath.Base(additionalFile))); err != nil {
							log.Fatal(err)
						}
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
				if err := utils.DownloadEveLive(certsDir, eveImageFile, eveArch, eveHV, eveTag, eveUefiTag, imageFormat); err != nil {
					log.Errorf("cannot download EVE: %s", err)
				} else {
					log.Infof("download EVE done: %s", eveImageFile)
					if devModel == defaults.DefaultRPIModel {
						log.Infof("Write file %s to sd (it is in raw format)", eveImageFile)
					}
				}
			} else {
				log.Infof("EVE already exists in dir: %s", filepath.Dir(eveImageFile))
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
	setupCmd.Flags().StringVarP(&adamDist, "adam-dist", "", "", "adam dist to start (required)")
	setupCmd.Flags().IntVarP(&adamPort, "adam-port", "", defaults.DefaultAdamPort, "adam dist to start")

	setupCmd.Flags().StringSliceVarP(&qemuFirmware, "eve-firmware", "", nil, "firmware path")
	setupCmd.Flags().StringVarP(&qemuConfigPath, "config-part", "", "", "path for config drive")
	setupCmd.Flags().StringVarP(&qemuDTBPath, "dtb-part", "", "", "path for device tree drive (for arm)")
	setupCmd.Flags().StringVarP(&eveImageFile, "image-file", "", "", "path for image drive (required)")
	setupCmd.Flags().StringVarP(&eveDist, "eve-dist", "", filepath.Join(currentPath, defaults.DefaultDist, defaults.DefaultEVEDist), "directory to save EVE")
	setupCmd.Flags().StringVarP(&eveRepo, "eve-repo", "", defaults.DefaultEveRepo, "EVE repo")
	setupCmd.Flags().StringVarP(&eveTag, "eve-tag", "", defaults.DefaultEVETag, "EVE tag")
	setupCmd.Flags().StringVarP(&eveUefiTag, "eve-uefi-tag", "", defaults.DefaultEVETag, "EVE UEFI tag")
	setupCmd.Flags().StringVarP(&eveArch, "eve-arch", "", runtime.GOARCH, "EVE arch")
	setupCmd.Flags().StringToStringVarP(&qemuHostFwd, "eve-hostfwd", "", defaults.DefaultQemuHostFwd, "port forward map")
	setupCmd.Flags().StringVarP(&qemuFileToSave, "qemu-config", "", filepath.Join(currentPath, defaults.DefaultDist, defaults.DefaultQemuFileToSave), "file to save qemu config")
	setupCmd.Flags().BoolVarP(&download, "download", "", true, "download EVE or build")
	setupCmd.Flags().StringVarP(&eveHV, "eve-hv", "", defaults.DefaultEVEHV, "hv of rootfs to use")

	setupCmd.Flags().StringVarP(&eserverImageDist, "image-dist", "", "", "image dist for eserver")
	setupCmd.Flags().StringVarP(&binDir, "bin-dist", "", filepath.Join(currentPath, defaults.DefaultDist, defaults.DefaultBinDist), "directory for binaries")
	setupCmd.Flags().BoolVarP(&force, "force", "", false, "force overwrite config file")
	setupCmd.Flags().BoolVarP(&dryRun, "dry-run", "", false, "")
	setupCmd.Flags().BoolVarP(&apiV1, "api-v1", "", true, "use v1 api")
}
