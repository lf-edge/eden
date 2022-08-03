package cmd

import (
	"compress/gzip"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"strings"

	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/eden"
	"github.com/lf-edge/eden/pkg/models"
	"github.com/lf-edge/eden/pkg/utils"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/crypto/pbkdf2"
	"golang.org/x/term"
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

	devModel  string
	modelFile string

	eveImageSizeMB int

	eveConfigDir string

	netboot   bool
	installer bool

	softserial    string
	zedcontrolURL string

	ipxeOverride string

	grubOptions []string

	eveDisks int
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
			hostFwd = viper.GetStringMapString("eve.hostfwd")
			qemuFileToSave = utils.ResolveAbsPath(viper.GetString("eve.qemu-config"))
			qemuCpus = viper.GetInt("eve.cpu")
			qemuMemory = viper.GetInt("eve.ram")
			eveImageSizeMB = viper.GetInt("eve.disk")
			eveDisks = viper.GetInt("eve.disks")
			//eserver
			eserverImageDist = utils.ResolveAbsPath(viper.GetString("eden.images.dist"))

			devModel = viper.GetString("eve.devmodel")

			ssid = viper.GetString("eve.ssid")
		}

		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		model, err := models.GetDevModelByName(devModel)
		if err != nil {
			log.Fatalf("GetDevModelByName: %s", err)
		}
		if netboot && installer {
			log.Fatal("Please use netboot or installer flag, not both")
		}
		if netboot || installer {
			if devModel != defaults.DefaultGeneralModel {
				log.Fatalf("Cannot use netboot for devmodel %s, please use general instead", devModel)
			}
		}
		if devModel == defaults.DefaultQemuModel {
			if _, err := os.Stat(qemuFileToSave); os.IsNotExist(err) {
				qemuDTBPathAbsolute := ""
				if qemuDTBPath != "" {
					qemuDTBPathAbsolute, err = filepath.Abs(qemuDTBPath)
					if err != nil {
						log.Fatal(err)
					}
				}
				var qemuFirmwareParam []string
				for _, line := range qemuFirmware {
					for _, el := range strings.Split(line, " ") {
						qemuFirmwareParam = append(qemuFirmwareParam, utils.ResolveAbsPath(el))
					}
				}
				var qemuDisksParam []string
				for ind := 0; ind < eveDisks; ind++ {
					diskFile := filepath.Join(filepath.Dir(eveImageFile), fmt.Sprintf("eve-disk-%d.qcow2", ind+1))
					if err := utils.CreateDisk(diskFile, "qcow2", uint64(eveImageSizeMB*1024*1024)); err != nil {
						log.Fatal(err)
					}
					qemuDisksParam = append(qemuDisksParam, diskFile)
				}
				settings := utils.QemuSettings{
					DTBDrive: qemuDTBPathAbsolute,
					Firmware: qemuFirmwareParam,
					Disks:    qemuDisksParam,
					MemoryMB: qemuMemory,
					CPUs:     qemuCpus,
				}
				conf, err := settings.GenerateQemuConfig()
				if err != nil {
					log.Fatal(err)
				}
				f, err := os.Create(qemuFileToSave)
				if err != nil {
					log.Fatal(err)
				}
				_, err = f.Write(conf)
				if err != nil {
					log.Fatal(err)
				}
				if err := f.Close(); err != nil {
					log.Fatal(err)
				}
				log.Infof("QEMU config file generated: %s", qemuFileToSave)
			} else {
				log.Debugf("QEMU config already exists: %s", qemuFileToSave)
			}
		}
		if _, err := os.Stat(filepath.Join(certsDir, "root-certificate.pem")); os.IsNotExist(err) {
			wifiPSK := ""
			if ssid != "" {
				fmt.Printf("Enter password for wifi %s: ", ssid)
				pass, _ := term.ReadPassword(0)
				wifiPSK = strings.ToLower(hex.EncodeToString(pbkdf2.Key(pass, []byte(ssid), 4096, 32, sha1.New)))
				fmt.Println()
			}
			if zedcontrolURL == "" {
				if err := eden.GenerateEveCerts(certsDir, certsDomain, certsIP, certsEVEIP, certsUUID, devModel, ssid, wifiPSK, grubOptions, apiV1); err != nil {
					log.Errorf("cannot GenerateEveCerts: %s", err)
				} else {
					log.Info("GenerateEveCerts done")
				}
			} else {
				if err := eden.PutEveCerts(certsDir, devModel, ssid, wifiPSK); err != nil {
					log.Errorf("cannot GenerateEveCerts: %s", err)
				} else {
					log.Info("GenerateEveCerts done")
				}
			}
		} else {
			log.Info("GenerateEveCerts done")
			log.Infof("Certs already exists in certs dir: %s", certsDir)
		}

		if zedcontrolURL == "" {
			if err := eden.GenerateEVEConfig(devModel, certsDir, certsDomain, certsEVEIP, adamPort, apiV1, softserial); err != nil {
				log.Errorf("cannot GenerateEVEConfig: %s", err)
			} else {
				log.Info("GenerateEVEConfig done")
			}
		} else {
			if err := eden.GenerateEVEConfig(devModel, certsDir, zedcontrolURL, "", 0, false, softserial); err != nil {
				log.Errorf("cannot GenerateEVEConfig: %s", err)
			} else {
				log.Info("GenerateEVEConfig done")
			}
		}
		if _, err := os.Lstat(configDir); !os.IsNotExist(err) {
			//put files from config folder to generated directory
			if err := utils.CopyFolder(utils.ResolveAbsPath(eveConfigDir), certsDir); err != nil {
				log.Errorf("CopyFolder: %s", err)
			}
		}
		imageFormat := model.DiskFormat()
		eveDesc := utils.EVEDescription{
			ConfigPath:  certsDir,
			Arch:        eveArch,
			HV:          eveHV,
			Registry:    eveRegistry,
			Tag:         eveTag,
			Format:      imageFormat,
			ImageSizeMB: eveImageSizeMB,
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
				if builedImage, builedAdditional, err = eden.MakeEveInRepo(eveDesc, eveDist); err != nil {
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
				log.Infof(model.DiskReadyMessage(), eveImageFile)
			} else {
				log.Infof("EVE already exists in dir: %s", eveDist)
			}
		} else {
			uefiDesc := utils.UEFIDescription{
				Registry: eveRegistry,
				Tag:      eveUefiTag,
				Arch:     eveArch,
			}
			imageTag, err := eveDesc.Image()
			if err != nil {
				log.Fatal(err)
			}
			if netboot {
				if err := utils.DownloadEveNetBoot(eveDesc, filepath.Dir(eveImageFile)); err != nil {
					log.Errorf("cannot download EVE: %s", err)
				} else {
					if err := eden.StartEServer(eserverPort, eserverImageDist, eserverForce, eserverTag); err != nil {
						log.Errorf("cannot start eserver: %s", err)
					} else {
						log.Infof("Eserver is running and accessible on port %d", eserverPort)
					}
					eServerIP := certsEVEIP
					eServerPort := viper.GetString("eden.eserver.port")
					server := &eden.EServer{
						EServerIP:   eServerIP,
						EServerPort: eServerPort,
					}
					// we should uncompress kernel for arm64
					if eveArch == "arm64" {
						// rename to temp file
						if err := os.Rename(filepath.Join(filepath.Dir(eveImageFile), "kernel"),
							filepath.Join(filepath.Dir(eveImageFile), "kernel.old")); err != nil {
							// probably naming changed, give up
							log.Warnf("Cannot rename kernel: %v", err)
						} else {
							r, err := os.Open(filepath.Join(filepath.Dir(eveImageFile), "kernel.old"))
							if err != nil {
								log.Fatalf("Open kernel.old: %v", err)
							}
							uncompressedStream, err := gzip.NewReader(r)
							if err != nil {
								// in case of non-gz rename back
								log.Errorf("gzip: NewReader failed: %v", err)
								if err := os.Rename(filepath.Join(filepath.Dir(eveImageFile), "kernel.old"),
									filepath.Join(filepath.Dir(eveImageFile), "kernel")); err != nil {
									log.Fatalf("Cannot rename kernel: %v", err)
								}
							} else {
								defer uncompressedStream.Close()
								out, err := os.Create(filepath.Join(filepath.Dir(eveImageFile), "kernel"))
								if err != nil {
									log.Fatalf("Cannot create file to save: %v", err)
								}
								if _, err := io.Copy(out, uncompressedStream); err != nil {
									log.Fatalf("Cannot copy to decompressed file: %v", err)
								}
								if err := out.Close(); err != nil {
									log.Fatalf("Cannot close file: %v", err)
								}
							}
						}
					}
					configPrefix := configName
					if configPrefix == defaults.DefaultContext {
						//in case of default context we use empty prefix to keep compatibility
						configPrefix = ""
					}
					items, _ := ioutil.ReadDir(filepath.Dir(eveImageFile))
					for _, item := range items {
						if !item.IsDir() && item.Name() != "ipxe.efi.cfg" {
							if _, err := eden.AddFileIntoEServer(server, filepath.Join(filepath.Dir(eveImageFile), item.Name()), configPrefix); err != nil {
								log.Fatalf("AddFileIntoEServer: %s", err)
							}
						}
					}
					ipxeFile := filepath.Join(filepath.Dir(eveImageFile), "ipxe.efi.cfg")
					ipxeFileBytes, err := ioutil.ReadFile(ipxeFile)
					if err != nil {
						log.Fatalf("Cannot read ipxe file: %v", err)
					}
					re := regexp.MustCompile("# set url .*")
					ipxeFileReplaced := re.ReplaceAll(ipxeFileBytes,
						[]byte(fmt.Sprintf("set url http://%s:%d/%s/", eServerIP, eserverPort, path.Join("eserver", configPrefix))))
					if softserial != "" {
						ipxeFileReplaced = []byte(strings.ReplaceAll(string(ipxeFileReplaced),
							"eve_soft_serial=${mac:hexhyp}",
							fmt.Sprintf("eve_soft_serial=%s", softserial)))
					}
					ipxeOverrideSlice := strings.Split(ipxeOverride, "||")
					if len(ipxeOverrideSlice) > 1 {
						fmt.Println(ipxeOverrideSlice)

						for i := 0; ; i += 2 {
							if i+1 >= len(ipxeOverrideSlice) {
								break
							}
							re := regexp.MustCompile(ipxeOverrideSlice[i])
							ipxeFileReplaced = re.ReplaceAll(ipxeFileReplaced, []byte(ipxeOverrideSlice[i+1]))
						}
					}
					_ = os.MkdirAll(filepath.Join(filepath.Dir(eveImageFile), "tftp"), 0777)
					ipxeConfigFile := filepath.Join(filepath.Dir(eveImageFile), "tftp", "ipxe.efi.cfg")
					_ = ioutil.WriteFile(ipxeConfigFile, ipxeFileReplaced, 0777)
					i, err := eden.AddFileIntoEServer(server, ipxeConfigFile, configPrefix)
					if err != nil {
						log.Fatalf("AddFileIntoEServer: %s", err)
					}
					log.Infof("download EVE done: %s", imageTag)
					log.Infof("Please use %s to boot your EVE via ipxe", ipxeConfigFile)
					log.Infof("ipxe.efi.cfg uploaded to eserver (http://%s:%s/%s). Use it to boot your EVE via network", eServerIP, eServerPort, i.FileName)
					log.Infof("EVE already exists: %s", filepath.Dir(eveImageFile))
				}
			} else if installer {
				if _, err := os.Lstat(eveImageFile); os.IsNotExist(err) {
					if err := utils.DownloadEveInstaller(eveDesc, eveImageFile); err != nil {
						log.Errorf("cannot download EVE: %s", err)
					} else {
						log.Infof("download EVE done: %s", imageTag)
						log.Infof(model.DiskReadyMessage(), eveImageFile)
					}
				} else {
					log.Infof("download EVE done: %s", imageTag)
					log.Infof("EVE already exists: %s", eveImageFile)
				}
			} else {
				if _, err := os.Lstat(eveImageFile); os.IsNotExist(err) {
					if err := utils.DownloadEveLive(eveDesc, uefiDesc, eveImageFile); err != nil {
						log.Errorf("cannot download EVE: %s", err)
					} else {
						log.Infof("download EVE done: %s", imageTag)
						log.Infof(model.DiskReadyMessage(), eveImageFile)
					}
				} else {
					log.Infof("download EVE done: %s", imageTag)
					log.Infof("EVE already exists: %s", eveImageFile)
				}
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
					fmt.Println("* for BASH/ZSH -- `source ~/.eden/activate.sh`")
					fmt.Println("* for TCSH -- `source ~/.eden/activate.csh`")
					fmt.Println("To deactivate them -- eden_deactivate")
				}
			}
			if zedcontrolURL != "" {
				log.Printf("Please use %s as Onboarding Key", defaults.OnboardUUID)
				if softserial != "" {
					log.Printf("use %s as Serial Number", softserial)
				}
				log.Printf("To onboard EVE onto %s", zedcontrolURL)
			}
		}
		// TODO: build SDN VM image
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
	setupCmd.Flags().StringToStringVarP(&hostFwd, "eve-hostfwd", "", defaults.DefaultQemuHostFwd, "port forward map")
	setupCmd.Flags().StringVarP(&qemuFileToSave, "qemu-config", "", "", "file to save qemu config")
	setupCmd.Flags().BoolVarP(&download, "download", "", true, "download EVE or build")
	setupCmd.Flags().StringVarP(&eveHV, "eve-hv", "", defaults.DefaultEVEHV, "hv of rootfs to use")

	setupCmd.Flags().StringVarP(&eserverImageDist, "image-dist", "", "", "image dist for eserver")
	setupCmd.Flags().StringVarP(&binDir, "bin-dist", "", filepath.Join(currentPath, defaults.DefaultDist, defaults.DefaultBinDist), "directory for binaries")
	setupCmd.Flags().BoolVarP(&force, "force", "", false, "force overwrite config file")
	setupCmd.Flags().BoolVarP(&dryRun, "dry-run", "", false, "")
	setupCmd.Flags().BoolVarP(&apiV1, "api-v1", "", true, "use v1 api")

	setupCmd.Flags().IntVar(&eveImageSizeMB, "image-size", defaults.DefaultEVEImageSize, "Image size of EVE in MB")

	setupCmd.Flags().StringVar(&eveConfigDir, "eve-config-dir", filepath.Join(currentPath, "eve-config-dir"), "directory with files to put into EVE`s conf directory during setup")

	setupCmd.Flags().BoolVar(&netboot, "netboot", false, "Setup for use with network boot")
	setupCmd.Flags().BoolVar(&installer, "installer", false, "Setup for create installer")
	setupCmd.Flags().StringVar(&softserial, "soft-serial", "", "Use provided serial instead of hardware one, please use chars and numbers here")
	setupCmd.Flags().StringVar(&zedcontrolURL, "zedcontrol", "", "Use provided zedcontrol domain instead of adam (as example: zedcloud.alpha.zededa.net)")

	setupCmd.Flags().StringVar(&ipxeOverride, "ipxe-override", "", "override lines inside ipxe, please use || as delimiter")
	setupCmd.Flags().StringArrayVar(&grubOptions, "grub-options", []string{}, "append lines to grub options")
	addSdnImageOpt(setupCmd)
}
