package openevec

import (
	"compress/gzip"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/lf-edge/eden/pkg/controller"
	"github.com/lf-edge/eden/pkg/controller/eflowlog"
	"github.com/lf-edge/eden/pkg/controller/einfo"
	"github.com/lf-edge/eden/pkg/controller/elog"
	"github.com/lf-edge/eden/pkg/controller/emetric"
	"github.com/lf-edge/eden/pkg/controller/types"
	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/eden"
	"github.com/lf-edge/eden/pkg/models"
	"github.com/lf-edge/eden/pkg/utils"
	"github.com/lf-edge/eve/api/go/flowlog"
	"github.com/lf-edge/eve/api/go/info"
	"github.com/lf-edge/eve/api/go/metrics"
	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"golang.org/x/crypto/pbkdf2"
	"golang.org/x/term"
)

func generateScripts(in string, out string, configFile string) {
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

func SetupEden(configName string, cfg EdenSetupArgs) error {

	configCheck(configName)

	if cfg.Runtime.Netboot && cfg.Runtime.Installer {
		log.Fatal("Please use netboot or installer flag, not both")
	}
	if cfg.Runtime.Netboot || cfg.Runtime.Installer {
		if cfg.Eve.DevModel != defaults.DefaultGeneralModel {
			log.Fatalf("Cannot use netboot for devmodel %s, please use general instead", cfg.Eve.DevModel)
		}
	}
	if cfg.Eve.DevModel == defaults.DefaultQemuModel {
		setupQemuConfig(cfg)
	}

	if cfg.Eve.CustomInstaller.Path == "" {
		setupConfigDir(cfg)
	}

	setupEve(cfg)

	setupEdenScripts(cfg)

	// Build Eden-SDN VM image unless the SDN is disabled.
	if isSdnEnabled(cfg.Sdn.Disable, cfg.Eve.Remote, cfg.Eve.DevModel) {
		setupSdn(cfg)
	}

	return nil
}

func setupQemuConfig(cfg EdenSetupArgs) {
	var err error
	if _, err = os.Stat(cfg.Eve.QemuFileToSave); err == nil || !os.IsNotExist(err) {
		log.Debugf("QEMU config already exists: %s", cfg.Eve.QemuFileToSave)
	}
	qemuDTBPathAbsolute := ""
	if cfg.Eve.QemuDTBPath != "" {
		qemuDTBPathAbsolute, err = filepath.Abs(cfg.Eve.QemuDTBPath)
		if err != nil {
			log.Fatal(err)
		}
	}
	var qemuFirmwareParam []string
	for _, line := range cfg.Eve.QemuFirmware {
		for _, el := range strings.Split(line, " ") {
			qemuFirmwareParam = append(qemuFirmwareParam, utils.ResolveAbsPath(el))
		}
	}
	if cfg.Eve.CustomInstaller.Path != "" && cfg.Eve.Disks == 0 {
		log.Fatal("EVE installer requires at least one disK")
	}
	var qemuDisksParam []string
	for ind := 0; ind < cfg.Eve.Disks; ind++ {
		diskFile := filepath.Join(filepath.Dir(cfg.Eve.ImageFile), fmt.Sprintf("eve-disk-%d.qcow2", ind+1))
		if err := utils.CreateDisk(diskFile, "qcow2", uint64(cfg.Eve.ImageSizeMB*1024*1024)); err != nil {
			log.Fatal(err)
		}
		qemuDisksParam = append(qemuDisksParam, diskFile)
	}
	settings := utils.QemuSettings{
		DTBDrive: qemuDTBPathAbsolute,
		Firmware: qemuFirmwareParam,
		Disks:    qemuDisksParam,
		MemoryMB: cfg.Eve.QemuMemory,
		CPUs:     cfg.Eve.QemuCpus,
	}
	conf, err := settings.GenerateQemuConfig()
	if err != nil {
		log.Fatal(err)
	}
	f, err := os.Create(cfg.Eve.QemuFileToSave)
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
	log.Infof("QEMU config file generated: %s", cfg.Eve.QemuFileToSave)
}

func setupEve(cfg EdenSetupArgs) error {
	model, err := models.GetDevModelByName(cfg.Eve.DevModel)
	if err != nil {
		return fmt.Errorf("GetDevModelByName: %s", err)
	}
	imageFormat := model.DiskFormat()
	eveDesc := utils.EVEDescription{
		ConfigPath:  cfg.Eden.CertsDir,
		Arch:        cfg.Eve.Arch,
		HV:          cfg.Eve.HV,
		Registry:    cfg.Eve.Registry,
		Tag:         cfg.Eve.Tag,
		Format:      imageFormat,
		ImageSizeMB: cfg.Eve.ImageSizeMB,
	}
	if cfg.Eve.CustomInstaller.Path != "" {
		// With installer image already prepared, install only UEFI.
		if imageFormat == "qcow2" {
			if err := utils.DownloadUEFI(eveDesc, filepath.Dir(cfg.Eve.ImageFile)); err != nil {
				log.Errorf("cannot download UEFI: %v", err)
			} else {
				log.Infof("download UEFI done")
			}
		}
		return nil
	}
	if !cfg.Eden.Download {
		if _, err := os.Lstat(cfg.Eve.ImageFile); os.IsNotExist(err) {
			if err := eden.CloneFromGit(cfg.Eve.Dist, cfg.Eve.Repo, cfg.Eve.Tag); err != nil {
				log.Errorf("cannot clone EVE: %s", err)
			} else {
				log.Info("clone EVE done")
			}
			builedImage := ""
			builedAdditional := ""
			if builedImage, builedAdditional, err = eden.MakeEveInRepo(eveDesc, cfg.Eve.Dist); err != nil {
				log.Errorf("cannot MakeEveInRepo: %s", err)
			} else {
				log.Info("MakeEveInRepo done")
			}
			if err = utils.CopyFile(builedImage, cfg.Eve.ImageFile); err != nil {
				return err
			}
			builedAdditionalSplitted := strings.Split(builedAdditional, ",")
			for _, additionalFile := range builedAdditionalSplitted {
				if additionalFile != "" {
					if err = utils.CopyFile(additionalFile, filepath.Join(filepath.Dir(cfg.Eve.ImageFile), filepath.Base(additionalFile))); err != nil {
						return err
					}
				}
			}
			log.Infof(model.DiskReadyMessage(), cfg.Eve.ImageFile)
		} else {
			log.Infof("EVE already exists in dir: %s", cfg.Eve.Dist)
		}
	} else { // download
		imageTag, err := eveDesc.Image()
		if err != nil {
			return err
		}
		if cfg.Runtime.Netboot {
			if err := utils.DownloadEveNetBoot(eveDesc, filepath.Dir(cfg.Eve.ImageFile)); err != nil {
				log.Errorf("cannot download EVE: %s", err)
			} else {
				if err := eden.StartEServer(cfg.Eden.Eserver.Port, cfg.Eden.Eserver.Images.EserverImageDist, cfg.Eden.Eserver.Force, cfg.Eden.Eserver.Tag); err != nil {
					log.Errorf("cannot start eserver: %s", err)
				} else {
					log.Infof("Eserver is running and accessible on port %d", cfg.Eden.Eserver.Port)
				}
				eServerIP := cfg.Adam.CertsEVEIP
				eServerPort := strconv.Itoa(cfg.Eden.Eserver.Port)
				server := &eden.EServer{
					EServerIP:   eServerIP,
					EServerPort: eServerPort,
				}
				// we should uncompress kernel for arm64
				if cfg.Eve.Arch == "arm64" {
					// rename to temp file
					if err := os.Rename(filepath.Join(filepath.Dir(cfg.Eve.ImageFile), "kernel"),
						filepath.Join(filepath.Dir(cfg.Eve.ImageFile), "kernel.old")); err != nil {
						// probably naming changed, give up
						log.Warnf("Cannot rename kernel: %v", err)
					} else {
						r, err := os.Open(filepath.Join(filepath.Dir(cfg.Eve.ImageFile), "kernel.old"))
						if err != nil {
							return fmt.Errorf("Open kernel.old: %v", err)
						}
						uncompressedStream, err := gzip.NewReader(r)
						if err != nil {
							// in case of non-gz rename back
							log.Errorf("gzip: NewReader failed: %v", err)
							if err := os.Rename(filepath.Join(filepath.Dir(cfg.Eve.ImageFile), "kernel.old"),
								filepath.Join(filepath.Dir(cfg.Eve.ImageFile), "kernel")); err != nil {
								return fmt.Errorf("Cannot rename kernel: %v", err)
							}
						} else {
							defer uncompressedStream.Close()
							out, err := os.Create(filepath.Join(filepath.Dir(cfg.Eve.ImageFile), "kernel"))
							if err != nil {
								return fmt.Errorf("Cannot create file to save: %v", err)
							}
							if _, err := io.Copy(out, uncompressedStream); err != nil {
								return fmt.Errorf("Cannot copy to decompressed file: %v", err)
							}
							if err := out.Close(); err != nil {
								return fmt.Errorf("Cannot close file: %v", err)
							}
						}
					}
				}
				configPrefix := cfg.ConfigName
				if configPrefix == defaults.DefaultContext {
					//in case of default context we use empty prefix to keep compatibility
					configPrefix = ""
				}
				items, _ := ioutil.ReadDir(filepath.Dir(cfg.Eve.ImageFile))
				for _, item := range items {
					if !item.IsDir() && item.Name() != "ipxe.efi.cfg" {
						if _, err := eden.AddFileIntoEServer(server, filepath.Join(filepath.Dir(cfg.Eve.ImageFile), item.Name()), configPrefix); err != nil {
							return fmt.Errorf("AddFileIntoEServer: %s", err)
						}
					}
				}
				ipxeFile := filepath.Join(filepath.Dir(cfg.Eve.ImageFile), "ipxe.efi.cfg")
				ipxeFileBytes, err := ioutil.ReadFile(ipxeFile)
				if err != nil {
					return fmt.Errorf("Cannot read ipxe file: %v", err)
				}
				re := regexp.MustCompile("# set url .*")
				ipxeFileReplaced := re.ReplaceAll(ipxeFileBytes,
					[]byte(fmt.Sprintf("set url http://%s:%d/%s/", eServerIP, eServerPort, path.Join("eserver", configPrefix))))
				if cfg.Runtime.Softserial != "" {
					ipxeFileReplaced = []byte(strings.ReplaceAll(string(ipxeFileReplaced),
						"eve_soft_serial=${mac:hexhyp}",
						fmt.Sprintf("eve_soft_serial=%s", cfg.Runtime.Softserial)))
				}
				ipxeOverrideSlice := strings.Split(cfg.Runtime.IpxeOverride, "||")
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
				_ = os.MkdirAll(filepath.Join(filepath.Dir(cfg.Eve.ImageFile), "tftp"), 0777)
				ipxeConfigFile := filepath.Join(filepath.Dir(cfg.Eve.ImageFile), "tftp", "ipxe.efi.cfg")
				_ = ioutil.WriteFile(ipxeConfigFile, ipxeFileReplaced, 0777)
				i, err := eden.AddFileIntoEServer(server, ipxeConfigFile, configPrefix)
				if err != nil {
					return fmt.Errorf("AddFileIntoEServer: %s", err)
				}
				log.Infof("download EVE done: %s", imageTag)
				log.Infof("Please use %s to boot your EVE via ipxe", ipxeConfigFile)
				log.Infof("ipxe.efi.cfg uploaded to eserver (http://%s:%s/%s). Use it to boot your EVE via network", eServerIP, eServerPort, i.FileName)
				log.Infof("EVE already exists: %s", filepath.Dir(cfg.Eve.ImageFile))
			}
		} else if cfg.Runtime.Installer {
			if _, err := os.Lstat(cfg.Eve.ImageFile); os.IsNotExist(err) {
				if err := utils.DownloadEveInstaller(eveDesc, cfg.Eve.ImageFile); err != nil {
					log.Errorf("cannot download EVE: %s", err)
				} else {
					log.Infof("download EVE done: %s", imageTag)
					log.Infof(model.DiskReadyMessage(), cfg.Eve.ImageFile)
				}
			} else {
				log.Infof("download EVE done: %s", imageTag)
				log.Infof("EVE already exists: %s", cfg.Eve.ImageFile)
			}
		} else { // download EVE live image
			if _, err := os.Lstat(cfg.Eve.ImageFile); os.IsNotExist(err) {
				if err := utils.DownloadEveLive(eveDesc, cfg.Eve.ImageFile); err != nil {
					log.Errorf("cannot download EVE: %s", err)
				} else {
					log.Infof("download EVE done: %s", imageTag)
					log.Infof(model.DiskReadyMessage(), cfg.Eve.ImageFile)
				}
				if imageFormat == "qcow2" {
					if err := utils.DownloadUEFI(eveDesc, filepath.Dir(cfg.Eve.ImageFile)); err != nil {
						log.Errorf("cannot download UEFI: %v", err)
					} else {
						log.Infof("download UEFI done")
					}
				}
			} else {
				log.Infof("download EVE done: %s", imageTag)
				log.Infof("EVE already exists: %s", cfg.Eve.ImageFile)
			}
		}
	}
	return nil
}

func setupEdenScripts(cfg EdenSetupArgs) {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Error(err)
	} else {
		cfgDir := home + "/.eden/"
		_, err = os.Stat(cfgDir)
		if err != nil {
			// Most likely running from inside of "eden test" which sets home directory
			// to "/no-home".
			fmt.Printf("Directory %s access error: %s\n",
				cfgDir, err)
		} else {
			shPath := viper.GetString("eden.root") + "/scripts/shell/"
			generateScripts(shPath+"activate.sh.tmpl",
				cfgDir+"activate.sh", cfg.ConfigName)
			generateScripts(shPath+"activate.csh.tmpl",
				cfgDir+"activate.csh", cfg.ConfigName)
			fmt.Println("To activate EDEN settings run:")
			fmt.Println("* for BASH/ZSH -- `source ~/.eden/activate.sh`")
			fmt.Println("* for TCSH -- `source ~/.eden/activate.csh`")
			fmt.Println("To deactivate them -- eden_deactivate")
		}
	}
}

func setupConfigDir(cfg EdenSetupArgs) {
	if _, err := os.Stat(filepath.Join(cfg.Eden.CertsDir, "root-certificate.pem")); os.IsNotExist(err) {
		wifiPSK := ""
		if cfg.Eve.Ssid != "" {
			fmt.Printf("Enter password for wifi %s: ", cfg.Eve.Ssid)
			pass, _ := term.ReadPassword(0)
			wifiPSK = strings.ToLower(hex.EncodeToString(pbkdf2.Key(pass, []byte(cfg.Eve.Ssid), 4096, 32, sha1.New)))
			fmt.Println()
		}
		if cfg.Runtime.ZedcontrolURL == "" {
			if err := eden.GenerateEveCerts(cfg.Eden.CertsDir, cfg.Adam.CertsDomain, cfg.Adam.CertsIP, cfg.Adam.CertsEVEIP, cfg.Eve.CertsUUID,
				cfg.Eve.DevModel, cfg.Eve.Ssid, wifiPSK, cfg.Runtime.GrubOptions, cfg.Adam.ApiV1); err != nil {
				log.Errorf("cannot GenerateEveCerts: %s", err)
			} else {
				log.Info("GenerateEveCerts done")
			}
		} else {
			if err := eden.PutEveCerts(cfg.Eden.CertsDir, cfg.Eve.DevModel, cfg.Eve.Ssid, wifiPSK); err != nil {
				log.Errorf("cannot GenerateEveCerts: %s", err)
			} else {
				log.Info("GenerateEveCerts done")
			}
		}
	} else {
		log.Info("GenerateEveCerts done")
		log.Infof("Certs already exists in certs dir: %s", cfg.Eden.CertsDir)
	}
	if cfg.Runtime.ZedcontrolURL == "" {
		err := eden.GenerateEVEConfig(cfg.Eve.DevModel, cfg.Eden.CertsDir, cfg.Adam.CertsDomain, cfg.Adam.CertsEVEIP,
			cfg.Adam.Port, cfg.Adam.ApiV1, cfg.Runtime.Softserial, cfg.Eve.BootstrapFile, isSdnEnabled(cfg.Sdn.Disable, cfg.Eve.Remote, cfg.Eve.DevModel))
		if err != nil {
			log.Errorf("cannot GenerateEVEConfig: %s", err)
		} else {
			log.Info("GenerateEVEConfig done")
		}
	} else {
		err := eden.GenerateEVEConfig(cfg.Eve.DevModel, cfg.Eden.CertsDir, cfg.Runtime.ZedcontrolURL, "", 0,
			false, cfg.Runtime.Softserial, cfg.Eve.BootstrapFile, isSdnEnabled(cfg.Sdn.Disable, cfg.Eve.Remote, cfg.Eve.DevModel))
		if err != nil {
			log.Errorf("cannot GenerateEVEConfig: %s", err)
		} else {
			log.Info("GenerateEVEConfig done")
		}
	}
	if _, err := os.Lstat(cfg.Runtime.ConfigDir); !os.IsNotExist(err) {
		//put files from config folder to generated directory
		if err := utils.CopyFolder(utils.ResolveAbsPath(cfg.Runtime.EveConfigDir), cfg.Eden.CertsDir); err != nil {
			log.Errorf("CopyFolder: %s", err)
		}
	}
	if cfg.Runtime.ZedcontrolURL != "" {
		log.Printf("Please use %s as Onboarding Key", defaults.OnboardUUID)
		if cfg.Runtime.Softserial != "" {
			log.Printf("use %s as Serial Number", cfg.Runtime.Softserial)
		}
		log.Printf("To onboard EVE onto %s", cfg.Runtime.ZedcontrolURL)
	}
}

func setupSdn(cfg EdenSetupArgs) error {
	if err := os.MkdirAll(cfg.Sdn.ConfigDir, 0777); err != nil {
		return fmt.Errorf("Failed to create directory for SDN config files: %v", err)
	}
	// Get Eden-SDN version.
	sdnVmSrcDir := filepath.Join(cfg.Sdn.SourceDir, "vm")
	cmdArgs := []string{"pkg", "show-tag", sdnVmSrcDir}
	output, err := exec.Command(cfg.Sdn.LinuxkitBin, cmdArgs...).Output()
	if err != nil {
		var stderr string
		if ee, ok := err.(*exec.ExitError); ok {
			stderr = string(ee.Stderr)
		} else {
			stderr = err.Error()
		}
		return fmt.Errorf("linuxkit pkg show-tag failed: %v", stderr)
	}
	sdnTag := strings.Split(string(output), ":")[1]
	sdnTag = strings.TrimSpace(sdnTag)
	// Build or preferably pull eden-sdn container.
	homeDir := filepath.Join(cfg.Eden.Root, "linuxkit-home")
	envVars := append(os.Environ(), fmt.Sprintf("HOME=%s", homeDir))
	sdnImage := fmt.Sprintf("%s:%s-%s", defaults.DefaultEdenSDNContainerRef, sdnTag, cfg.Eve.Arch)
	err = utils.PullImage(sdnImage)
	if err != nil {
		log.Warnf("failed to pull eden-sdn image (%s, err: %v), "+
			"trying to build locally instead...", sdnImage, err)
		platform := fmt.Sprintf("%s/%s", cfg.Eve.QemuOS, cfg.Eve.Arch)
		cmdArgs = []string{"pkg", "build", "--force", "--platforms", platform,
			"--docker", "--build-yml", "build.yml", sdnVmSrcDir}
		err := utils.RunCommandForegroundWithOpts(cfg.Sdn.LinuxkitBin, cmdArgs,
			utils.SetCommandEnvVars(envVars))
		if err != nil {
			return fmt.Errorf("Failed to build eden-sdn container: %v", err)
		}
	}
	// Build Eden-SDN VM qcow2 image.
	imageDir := filepath.Dir(cfg.Sdn.ImageFile)
	_ = os.MkdirAll(imageDir, 0777)
	vmYmlIn, err := ioutil.ReadFile(filepath.Join(cfg.Sdn.SourceDir, "sdn-vm.yml.in"))
	if err != nil {
		return fmt.Errorf("Failed to read eden-sdn vm.yml.in: %v", err)
	}
	vmYml := strings.ReplaceAll(string(vmYmlIn), "SDN_TAG", sdnTag)
	cmdArgs = []string{"build", "--arch", cfg.Eve.Arch, "--format", "qcow2-efi",
		"--docker", "--dir", imageDir, "--name", "sdn", "-"}
	err = utils.RunCommandForegroundWithOpts(cfg.Sdn.LinuxkitBin, cmdArgs,
		utils.SetCommandStdin(vmYml), utils.SetCommandEnvVars(envVars))
	if err != nil {
		return fmt.Errorf("Failed to build eden-sdn VM image: %v", err)
	}
	// This image filename is given by linuxkit.
	imageFilename := filepath.Join(imageDir, "sdn-efi.qcow2")
	if imageFilename != cfg.Sdn.ImageFile {
		err = os.Rename(imageFilename, cfg.Sdn.ImageFile)
		if err != nil {
			return fmt.Errorf("Failed to rename eden-sdn VM image to requested "+
				"filepath %s: %v", cfg.Sdn.ImageFile, err)
		}
	}
	// Build UEFI for SDN VM
	eveDesc := utils.EVEDescription{
		ConfigPath:  cfg.Eden.CertsDir,
		Arch:        cfg.Eve.Arch,
		HV:          cfg.Eve.HV,
		Registry:    cfg.Eve.Registry,
		Tag:         cfg.Eve.Tag,
		Format:      cfg.Runtime.ImageFormat,
		ImageSizeMB: cfg.Eve.ImageSizeMB,
	}
	if err := utils.DownloadUEFI(eveDesc, imageDir); err != nil {
		log.Errorf("cannot download UEFI (for SDN): %v", err)
	} else {
		log.Infof("download UEFI (for SDN) done")
	}
	return nil
}

func EdenClean(cfg EdenSetupArgs, configSaved, configName string) error {
	if cfg.Runtime.CurrentContext {
		log.Info("Cleanup current context")
		// we need to delete information about EVE from adam
		if err := StartRedis(cfg); err != nil {
			log.Errorf("cannot start redis: %s", err)
		} else {
			log.Infof("Redis is running and accessible on port %d", cfg.Redis.Port)
		}
		if err := StartAdam(cfg); err != nil {
			log.Errorf("cannot start adam: %s", err)
		} else {
			log.Infof("Adam is running and accessible on port %d", cfg.Adam.Port)
		}
		if err := eden.CleanContext(cfg.Eve.Dist, cfg.Eden.CertsDir, filepath.Dir(cfg.Eve.ImageFile), cfg.Eve.Pid, cfg.Eve.CertsUUID,
			cfg.Sdn.PidFile, cfg.Runtime.VmName, configSaved, cfg.Eve.Remote); err != nil {
			log.Fatalf("cannot CleanContext: %s", err)
		}
	} else {
		if err := eden.CleanEden(cfg.Eve.Dist, cfg.Adam.Dist, cfg.Eden.CertsDir, filepath.Dir(cfg.Eve.ImageFile),
			cfg.Eden.Images.EserverImageDist, cfg.Redis.Dist, cfg.Registry.Dist, cfg.Runtime.ConfigDir, cfg.Eve.Pid,
			cfg.Sdn.PidFile, configSaved, cfg.Eve.Remote, cfg.Eve.DevModel, cfg.Runtime.VmName); err != nil {
			log.Fatalf("cannot CleanEden: %s", err)
		}
	}
	log.Infof("CleanEden done")
	return nil
}

func EdenInfo(cfg *EdenSetupArgs, outputFormat types.OutputFormat, args []string) error {
	ctrl, err := controller.CloudPrepare()
	if err != nil {
		return fmt.Errorf("CloudPrepare: %s", err)
	}
	devFirst, err := ctrl.GetDeviceCurrent()
	if err != nil {
		return fmt.Errorf("GetDeviceCurrent error: %s", err)
	}
	devUUID := devFirst.GetID()
	q := make(map[string]string)
	for _, a := range args[0:] {
		s := strings.Split(a, ":")
		q[s[0]] = s[1]
	}

	handleInfo := func(im *info.ZInfoMsg) bool {
		if cfg.Runtime.PrintFields == nil {
			einfo.ZInfoPrn(im, outputFormat)
		} else {
			einfo.ZInfoPrintFiltered(im, cfg.Runtime.PrintFields).Print()
		}
		return false
	}
	if cfg.Runtime.InfoTail > 0 {
		if err = ctrl.InfoChecker(devUUID, q, handleInfo, einfo.InfoTail(cfg.Runtime.InfoTail), 0); err != nil {
			return fmt.Errorf("InfoChecker: %s", err)
		}
	} else {
		if cfg.Runtime.Follow {
			if err = ctrl.InfoChecker(devUUID, q, handleInfo, einfo.InfoNew, 0); err != nil {
				return fmt.Errorf("InfoChecker: %s", err)
			}
		} else {
			if err = ctrl.InfoLastCallback(devUUID, q, handleInfo); err != nil {
				return fmt.Errorf("InfoChecker: %s", err)
			}
		}
	}
	return nil
}

func EdenLog(cfg *EdenSetupArgs, outputFormat types.OutputFormat, args []string) error {
	ctrl, err := controller.CloudPrepare()
	if err != nil {
		return fmt.Errorf("CloudPrepare: %s", err)
	}
	devFirst, err := ctrl.GetDeviceCurrent()
	if err != nil {
		return fmt.Errorf("GetDeviceCurrent error: %s", err)
	}
	devUUID := devFirst.GetID()

	q := make(map[string]string)

	for _, a := range args[0:] {
		s := strings.Split(a, ":")
		q[s[0]] = s[1]
	}

	handleFunc := func(le *elog.FullLogEntry) bool {
		if cfg.Runtime.PrintFields == nil {
			elog.LogPrn(le, outputFormat)
		} else {
			elog.LogItemPrint(le, outputFormat, cfg.Runtime.PrintFields).Print()
		}
		return false
	}

	if cfg.Runtime.LogTail > 0 {
		if err = ctrl.LogChecker(devUUID, q, handleFunc, elog.LogTail(cfg.Runtime.LogTail), 0); err != nil {
			return fmt.Errorf("LogChecker: %s", err)
		}
	} else {
		if cfg.Runtime.Follow {
			// Monitoring of new files
			if err = ctrl.LogChecker(devUUID, q, handleFunc, elog.LogNew, 0); err != nil {
				return fmt.Errorf("LogChecker: %s", err)
			}
		} else {
			if err = ctrl.LogLastCallback(devUUID, q, handleFunc); err != nil {
				return fmt.Errorf("LogChecker: %s", err)
			}
		}
	}
	return nil
}

func EdenStat(cfg *EdenSetupArgs, outputFormat types.OutputFormat, args []string) error {
	ctrl, err := controller.CloudPrepare()
	if err != nil {
		return fmt.Errorf("CloudPrepare: %s", err)
	}
	devFirst, err := ctrl.GetDeviceCurrent()
	if err != nil {
		return fmt.Errorf("GetDeviceCurrent error: %s", err)
	}
	devUUID := devFirst.GetID()

	q := make(map[string]string)

	for _, a := range args[0:] {
		s := strings.Split(a, ":")
		q[s[0]] = s[1]
	}

	handleFunc := func(le *flowlog.FlowMessage) bool {
		if cfg.Runtime.PrintFields == nil {
			eflowlog.FlowLogPrn(le, outputFormat)
		} else {
			eflowlog.FlowLogItemPrint(le, cfg.Runtime.PrintFields).Print()
		}
		return false
	}

	if cfg.Runtime.LogTail > 0 {
		if err = ctrl.FlowLogChecker(devUUID, q, handleFunc, eflowlog.FlowLogTail(cfg.Runtime.LogTail), 0); err != nil {
			return fmt.Errorf("FlowLogChecker: %s", err)
		}
	} else {
		if cfg.Runtime.Follow {
			// Monitoring of new files
			if err = ctrl.FlowLogChecker(devUUID, q, handleFunc, eflowlog.FlowLogNew, 0); err != nil {
				return fmt.Errorf("FlowLogChecker: %s", err)
			}
		} else {
			if err = ctrl.FlowLogLastCallback(devUUID, q, handleFunc); err != nil {
				return fmt.Errorf("FlowLogLastCallback: %s", err)
			}
		}
	}
	return nil
}

func EdenMetric(cfg *EdenSetupArgs, outputFormat types.OutputFormat, args []string) error {
	ctrl, err := controller.CloudPrepare()
	if err != nil {
		return fmt.Errorf("CloudPrepare: %s", err)
	}
	devFirst, err := ctrl.GetDeviceCurrent()
	if err != nil {
		return fmt.Errorf("GetDeviceCurrent error: %s", err)
	}
	devUUID := devFirst.GetID()

	q := make(map[string]string)

	for _, a := range args[0:] {
		s := strings.Split(a, ":")
		q[s[0]] = s[1]
	}

	handleFunc := func(le *metrics.ZMetricMsg) bool {
		if cfg.Runtime.PrintFields == nil {
			emetric.MetricPrn(le, outputFormat)
		} else {
			emetric.MetricItemPrint(le, cfg.Runtime.PrintFields).Print()
		}
		return false
	}

	if cfg.Runtime.MetricTail > 0 {
		if err = ctrl.MetricChecker(devUUID, q, handleFunc, emetric.MetricTail(cfg.Runtime.MetricTail), 0); err != nil {
			return fmt.Errorf("MetricChecker: %s", err)
		}
	} else {
		if cfg.Runtime.Follow {
			// Monitoring of new files
			if err = ctrl.MetricChecker(devUUID, q, handleFunc, emetric.MetricNew, 0); err != nil {
				return fmt.Errorf("MetricChecker: %s", err)
			}
		} else {
			if err = ctrl.MetricLastCallback(devUUID, q, handleFunc); err != nil {
				return fmt.Errorf("MetricChecker: %s", err)
			}
		}
	}
	return nil
}

func EdenExport(tarFile string, cfg *EdenSetupArgs) error {
	changer := &adamChanger{}
	// we need to obtain information about EVE from Adam
	if err := eden.StartRedis(cfg.Redis.Port, cfg.Redis.Dist, false, cfg.Redis.Tag); err != nil {
		return fmt.Errorf("cannot start redis: %s", err)
	} else {
		log.Infof("Redis is running and accessible on port %d", cfg.Redis.Port)
	}
	if err := eden.StartAdam(cfg.Adam.Port, cfg.Adam.Dist, false, cfg.Adam.Tag, cfg.Adam.Redis.RemoteURL, cfg.Adam.ApiV1); err != nil {
		return fmt.Errorf("cannot start adam: %s", err)
	} else {
		log.Infof("Adam is running and accessible on port %d", cfg.Adam.Port)
	}
	ctrl, err := changer.getController()
	if err != nil {
		return fmt.Errorf("getControllerAndDev: %s", err)
	}
	dev, err := ctrl.GetDeviceCurrent()
	if err == nil {
		deviceCert, err := ctrl.GetDeviceCert(dev)
		if err != nil {
			log.Warn(err)
		} else {
			if err = ioutil.WriteFile(ctrl.GetVars().EveDeviceCert, deviceCert.Cert, 0777); err != nil {
				log.Warn(err)
			}
		}
	} else {
		log.Info("Device not registered, will not save device cert")
	}
	edenDir, err := utils.DefaultEdenDir()
	if err != nil {
		return err
	}
	files := []utils.FileToSave{
		{Location: cfg.Eden.CertsDir, Destination: filepath.Join("dist", filepath.Base(cfg.Eden.CertsDir))},
		{Location: utils.ResolveAbsPath(defaults.DefaultCertsDist), Destination: filepath.Join("dist", defaults.DefaultCertsDist)},
		{Location: edenDir, Destination: "eden"},
	}
	if err := utils.CreateTarGz(tarFile, files); err != nil {
		log.Fatal(err)
	}
	log.Infof("Export Eden done")
	return nil
}

func EdenImport(tarFile string, rewriteRoot bool, cfg *EdenSetupArgs) error {
	edenDir, err := utils.DefaultEdenDir()
	if err != nil {
		return err
	}
	files := []utils.FileToSave{
		{Location: filepath.Join("dist", filepath.Base(cfg.Eden.CertsDir)), Destination: cfg.Eden.CertsDir},
		{Location: filepath.Join("dist", defaults.DefaultCertsDist), Destination: utils.ResolveAbsPath(defaults.DefaultCertsDist)},
		{Location: "eden", Destination: edenDir},
	}
	if err := utils.UnpackTarGz(tarFile, files); err != nil {
		return err
	}
	if rewriteRoot {
		// we need to rewrite eden root to match with local
		viperLoaded, err := utils.LoadConfigFile(cfg.ConfigFile)
		if err != nil {
			return fmt.Errorf("error reading config: %s", err.Error())
		}
		if viperLoaded {
			if cfg.Eden.Root != viper.GetString("eden.root") {
				viper.Set("eve.root", cfg.Eden.Root)
				if err = utils.GenerateConfigFileFromViper(); err != nil {
					return fmt.Errorf("error writing config: %s", err)
				}
			}
		}
	}
	// we need to put information about EVE into Adam
	if err := eden.StartRedis(cfg.Redis.Port, cfg.Redis.Dist, false, cfg.Redis.Tag); err != nil {
		log.Errorf("cannot start redis: %s", err)
	} else {
		log.Infof("Redis is running and accessible on port %d", cfg.Redis.Port)
	}
	if err := eden.StartAdam(cfg.Adam.Port, cfg.Adam.Dist, false, cfg.Adam.Tag, cfg.Adam.Redis.RemoteURL, cfg.Adam.ApiV1); err != nil {
		log.Errorf("cannot start adam: %s", err)
	} else {
		log.Infof("Adam is running and accessible on port %d", cfg.Adam.Port)
	}
	changer := &adamChanger{}
	ctrl, err := changer.getController()
	if err != nil {
		return err
	}
	devUUID, err := ctrl.DeviceGetByOnboard(ctrl.GetVars().EveCert)
	if err != nil {
		log.Debug(err)
	}
	if devUUID == uuid.Nil {
		if _, err := os.Stat(ctrl.GetVars().EveDeviceCert); os.IsNotExist(err) {
			log.Warnf("No device cert %s, you device was not registered", ctrl.GetVars().EveDeviceCert)
		} else {
			if _, err := os.Stat(ctrl.GetVars().EveCert); os.IsNotExist(err) {
				return fmt.Errorf("No onboard cert in %s, you need to run 'eden setup' first", ctrl.GetVars().EveCert)
			}
			deviceCert, err := ioutil.ReadFile(ctrl.GetVars().EveDeviceCert)
			if err != nil {
				return err
			}
			onboardCert, err := ioutil.ReadFile(ctrl.GetVars().EveCert)
			if err != nil {
				log.Warn(err)
			}
			dc := types.DeviceCert{
				Cert:   deviceCert,
				Serial: ctrl.GetVars().EveSerial,
			}
			if onboardCert != nil {
				dc.Onboard = onboardCert
			}
			err = ctrl.UploadDeviceCert(dc)
			if err != nil {
				return err
			}
		}
		log.Info("You need to run 'eden eve onboard")
	} else {
		log.Info("Device already exists")
	}

	return nil
}
