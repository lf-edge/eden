package cmd

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"github.com/lf-edge/eden/pkg/eden"
	"io/ioutil"
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

	eveRegistry string

	devModel string

	eveImageSizeMB int
)

func generateScripts(in string, out string) {
	tmpl, err := ioutil.ReadFile(in)
	if err != nil {
		log.Fatal(err)
	}
	script, err := utils.RenderTemplate(configFile, string(tmpl))
	if err != nil {
		log.Fatal(err)
	}
	err = ioutil.WriteFile(out, []byte(script), 0644)
	if err != nil {
		log.Fatal(err)
	}
}

func configCheck() {
	configSaved = utils.ResolveAbsPath(fmt.Sprintf("%s-%s", configName, defaults.DefaultConfigSaved))

	abs, err := filepath.Abs(configSaved)
	if err != nil {
		log.Fatalf("fail in reading filepath: %s\n", err.Error())
	}

	if _, err = os.Lstat(abs); os.IsNotExist(err) {
		if err = utils.CopyFile(configFile, abs); err != nil {
			log.Fatalf("copying fail %s\n", err.Error())
		}
	} else {

		viperLoaded, err := utils.LoadConfigFile(abs)
		if err != nil {
			log.Fatalf("error reading config %s: %s\n", abs, err.Error())
		}
		if viperLoaded {
			confOld := viper.AllSettings()

			if _, err = utils.LoadConfigFile(configFile); err != nil {
				log.Fatalf("error reading config %s: %s", configFile, err.Error())
			}

			confCur := viper.AllSettings()

			if reflect.DeepEqual(confOld, confCur) {
				log.Infof("Config file %s is the same as %s\n", configFile, configSaved)
			} else {
				log.Fatalf("The current configuration file %s is different from the saved %s. You can fix this with the commands 'eden config clean' and 'eden config add/set/edit'.\n", configFile, abs)
			}
		} else {
			/* Incorrect saved config -- just rewrite by current */
			if err = utils.CopyFile(configFile, abs); err != nil {
				log.Fatalf("copying fail %s\n", err.Error())
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
			eveRegistry = viper.GetString("eve.registry")
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
			if err := eden.GenerateEveCerts(command, configName, certsDir, certsDomain, certsIP, certsEVEIP, certsUUID, ssid, wifiPSK); err != nil {
				log.Errorf("cannot GenerateEveCerts: %s", err)
			} else {
				log.Info("GenerateEveCerts done")
			}
		} else {
			log.Info("GenerateEveCerts done")
			log.Infof("Certs already exists in certs dir: %s", certsDir)
		}
		if err := eden.GenerateEVEConfig(certsDir, certsDomain, certsEVEIP, adamPort, apiV1); err != nil {
			log.Errorf("cannot GenerateEVEConfig: %s", err)
		} else {
			log.Info("GenerateEVEConfig done")
		}
		var imageFormat string
		switch devModel {
		case defaults.DefaultRPIModel:
			imageFormat = "raw"
		case defaults.DefaultGCPModel:
			imageFormat = "gcp"
		case defaults.DefaultEVEModel:
			imageFormat = "qcow2"
		default:
			log.Fatalf("Unsupported dev model %s", devModel)
		}
		if !download {
			if _, err := os.Lstat(eveImageFile); os.IsNotExist(err) {
				if err := eden.CloneFromGit(eveDist, eveRepo, eveTag); err != nil {
					log.Errorf("cannot clone EVE: %s", err)
				} else {
					log.Info("clone EVE done")
				}
				builedImage := ""
				builedAdditional := ""
				if builedImage, builedAdditional, err = eden.MakeEveInRepo(eveDist, certsDir, eveArch, eveHV, imageFormat, false); err != nil {
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

				switch devModel {
				case defaults.DefaultRPIModel:
					log.Infof("Write file %s to sd (it is in raw format)", eveImageFile)
				case defaults.DefaultGCPModel:
					log.Infof("Upload %s to gcp and run", eveImageFile)
				}
			} else {
				log.Infof("EVE already exists in dir: %s", eveDist)
			}
		} else {
			if _, err := os.Lstat(eveImageFile); os.IsNotExist(err) {
				eveDesc := utils.EVEDescription{
					ConfigPath:  certsDir,
					Arch:        eveArch,
					HV:          eveHV,
					Registry:    eveRegistry,
					Tag:         eveTag,
					Format:      imageFormat,
					ImageSizeMB: eveImageSizeMB,
				}
				uefiDesc := utils.UEFIDescription{
					Registry: eveRegistry,
					Tag:      eveUefiTag,
					Arch:     eveArch,
				}
				if err := utils.DownloadEveLive(eveDesc, uefiDesc, eveImageFile); err != nil {
					log.Errorf("cannot download EVE: %s", err)
				} else {
					log.Infof("download EVE done: %s", eveImageFile)
					switch devModel {
					case defaults.DefaultRPIModel:
						log.Infof("Write file %s to sd (it is in raw format)", eveImageFile)
					case defaults.DefaultGCPModel:
						log.Infof("Upload %s to gcp and run", eveImageFile)
					}
				}
			} else {
				log.Infof("download EVE done: %s", eveImageFile)
				log.Infof("EVE already exists in dir: %s", filepath.Dir(eveImageFile))
			}

			home, err := os.UserHomeDir()
			if err != nil {
				log.Error(err)
			} else {
				cfgDir := home + "/.eden/"
				_, err = os.Stat(cfgDir)
				if err != nil {
					fmt.Printf("Directory %s access error: %s\n",
						cfgDir, err)
				} else {
					shPath := viper.GetString("eden.root") + "/scripts/shell/"
					generateScripts(shPath+"activate.sh.tmpl",
						cfgDir+"activate.sh")
					generateScripts(shPath+"activate.csh.tmpl",
						cfgDir+"activate.csh")
					fmt.Println("To activate EDEN settings run:")
					fmt.Println("* for BASH -- `source ~/.eden/activate.sh`")
					fmt.Println("* for TCSH -- `source ~/.eden/activate.csh`")
					fmt.Println("To deactivate them -- eden_deactivate")
				}
			}
		}
	},
}

func setupInit() {
	currentPath, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	setupCmd.Flags().StringVarP(&certsDir, "certs-dist", "o", "", "directory with certs")
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
	setupCmd.Flags().StringVarP(&eveDist, "eve-dist", "", "", "directory to save EVE")
	setupCmd.Flags().StringVarP(&eveRepo, "eve-repo", "", defaults.DefaultEveRepo, "EVE repo")
	setupCmd.Flags().StringVarP(&eveRegistry, "eve-registry", "", defaults.DefaultEveRegistry, "EVE registry")
	setupCmd.Flags().StringVarP(&eveTag, "eve-tag", "", defaults.DefaultEVETag, "EVE tag")
	setupCmd.Flags().StringVarP(&eveUefiTag, "eve-uefi-tag", "", defaults.DefaultEVETag, "EVE UEFI tag")
	setupCmd.Flags().StringVarP(&eveArch, "eve-arch", "", runtime.GOARCH, "EVE arch")
	setupCmd.Flags().StringToStringVarP(&qemuHostFwd, "eve-hostfwd", "", defaults.DefaultQemuHostFwd, "port forward map")
	setupCmd.Flags().StringVarP(&qemuFileToSave, "qemu-config", "", "", "file to save qemu config")
	setupCmd.Flags().BoolVarP(&download, "download", "", true, "download EVE or build")
	setupCmd.Flags().StringVarP(&eveHV, "eve-hv", "", defaults.DefaultEVEHV, "hv of rootfs to use")

	setupCmd.Flags().StringVarP(&eserverImageDist, "image-dist", "", "", "image dist for eserver")
	setupCmd.Flags().StringVarP(&binDir, "bin-dist", "", filepath.Join(currentPath, defaults.DefaultDist, defaults.DefaultBinDist), "directory for binaries")
	setupCmd.Flags().BoolVarP(&force, "force", "", false, "force overwrite config file")
	setupCmd.Flags().BoolVarP(&dryRun, "dry-run", "", false, "")
	setupCmd.Flags().BoolVarP(&apiV1, "api-v1", "", true, "use v1 api")

	setupCmd.Flags().IntVar(&eveImageSizeMB, "image-size", defaults.DefaultEVEImageSize, "Image size of EVE in MB")
}
