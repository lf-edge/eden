package cmd

import (
	"fmt"
	"github.com/lf-edge/eden/pkg/utils"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

var (
	eveDist     string
	eveBaseDist string
	eveRepo     string
	eveBaseTag  string
	download    bool
	binDir      string
	dockerYML   string
	vmYML       string
	force       bool
	rootDir     string
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "config harness",
	Long:  `Config harness.`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		assingCobraToViper(cmd)
		viperLoaded, err := utils.LoadConfigFile(config)
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
			adamPort = viper.GetString("adam.port")
			adamDist = utils.ResolveAbsPath(viper.GetString("adam.dist"))
			certsDomain = viper.GetString("adam.domain")
			certsIP = viper.GetString("adam.ip")
			//eve
			qemuFirmware = viper.GetStringSlice("eve.firmware")
			qemuConfigPath = utils.ResolveAbsPath(viper.GetString("eve.config-part"))
			qemuDTBPath = utils.ResolveAbsPath(viper.GetString("eve.dtb-part"))
			eveImageFile = utils.ResolveAbsPath(viper.GetString("eve.image-file"))
			certsUUID = viper.GetString("eve.uuid")
			eveDist = utils.ResolveAbsPath(viper.GetString("eve.dist"))
			eveBaseDist = utils.ResolveAbsPath(viper.GetString("eve.base-dist"))
			eveRepo = viper.GetString("eve.repo")
			eveTag = viper.GetString("eve.tag")
			eveBaseTag = viper.GetString("eve.base-tag")
			eveHV = viper.GetString("eve.hv")
			qemuHostFwd = viper.GetStringMapString("eve.hostfwd")
			qemuFileToSave = utils.ResolveAbsPath(viper.GetString("eve.qemu-config"))
			//eserver
			eserverImageDist = utils.ResolveAbsPath(viper.GetString("eden.images.dist"))
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		command, err := os.Executable()
		if err != nil {
			log.Fatalf("cannot obtain executable path: %s", err)
		}
		if _, err := os.Stat(filepath.Join(certsDir, "server.pem")); os.IsNotExist(err) {
			if err := utils.GenerateEveCerts(command, certsDir, certsDomain, certsIP, certsUUID); err != nil {
				log.Errorf("cannot GenerateEveCerts: %s", err)
			} else {
				log.Info("GenerateEveCerts done")
			}
		} else {
			log.Infof("Certs already exists in certs dir: %s", certsDir)
		}
		if _, err := os.Stat(filepath.Join(adamDist, "run", "config", "server.pem")); os.IsNotExist(err) {
			if err := utils.CopyCertsToAdamConfig(certsDir, certsDomain, certsIP, adamPort, adamDist); err != nil {
				log.Errorf("cannot CopyCertsToAdamConfig: %s", err)
			} else {
				log.Info("CopyCertsToAdamConfig done")
			}
		} else {
			log.Infof("Certs already exists in adam dir: %s", certsDir)
		}
		if _, err := os.Stat(filepath.Join(adamDist, "run", "config", "server.pem")); os.IsNotExist(err) {
			if err := utils.CopyCertsToAdamConfig(certsDir, certsDomain, certsIP, adamPort, adamDist); err != nil {
				log.Errorf("cannot CopyCertsToAdamConfig: %s", err)
			} else {
				log.Info("CopyCertsToAdamConfig done")
			}
		} else {
			log.Infof("Certs already exists in adam dir: %s", certsDir)
		}
		if !download {
			if _, err := os.Lstat(eveImageFile); os.IsNotExist(err) {
				if err := utils.CloneFromGit(eveDist, eveRepo, eveTag); err != nil {
					log.Errorf("cannot clone EVE: %s", err)
				} else {
					log.Info("clone EVE done")
				}
				if err := utils.MakeEveInRepo(eveDist, adamDist, eveArch, eveHV, false); err != nil {
					log.Errorf("cannot MakeEveInRepo: %s", err)
				} else {
					log.Info("MakeEveInRepo done")
				}
			} else {
				log.Infof("EVE already exists in dir: %s", eveDist)

			}
			if _, err := os.Stat(eveBaseDist); os.IsNotExist(err) {
				if err := utils.CloneFromGit(eveBaseDist, eveRepo, eveBaseTag); err != nil {
					log.Errorf("cannot clone BASE EVE: %s", err)
				} else {
					log.Info("clone BASE EVE done")
				}
				if err := utils.MakeEveInRepo(eveBaseDist, adamDist, eveArch, eveHV, true); err != nil {
					log.Errorf("cannot MakeEveInRepo base: %s", err)
				} else {
					log.Info("MakeEveInRepo base done")
				}
			} else {
				log.Infof("BASE EVE already exists in dir: %s", eveBaseDist)
			}
		} else {
			if _, err := os.Lstat(eveImageFile); os.IsNotExist(err) {
				if err := utils.DownloadEveFormDocker(command, eveDist, eveArch, eveTag, false); err != nil {
					log.Errorf("cannot download EVE: %s", err)
				} else {
					log.Info("download EVE done")
				}
				if err := utils.ChangeConfigPartAndRootFs(command, eveDist, adamDist, eveArch, eveHV); err != nil {
					log.Errorf("cannot ChangeConfigPartAndRootFs EVE: %s", err)
				} else {
					log.Info("ChangeConfigPartAndRootFs EVE done")
				}
			} else {
				log.Infof("EVE already exists in dir: %s", eveDist)
			}
			if _, err := os.Stat(eveBaseDist); os.IsNotExist(err) {
				if err := utils.DownloadEveFormDocker(command, eveBaseDist, eveArch, eveBaseTag, true); err != nil {
					log.Errorf("cannot download Base EVE: %s", err)
				} else {
					log.Info("download Base EVE done")
				}
				if err := utils.ChangeConfigPartAndRootFs(command, eveBaseDist, adamDist, eveArch, eveHV); err != nil {
					log.Errorf("cannot ChangeConfigPartAndRootFs Base EVE: %s", err)
				} else {
					log.Info("ChangeConfigPartAndRootFs Base EVE done")
				}
			} else {
				log.Infof("Base EVE already exists in dir: %s", eveBaseDist)
			}
		}
		if err = utils.CopyFileNotExists(filepath.Join(eveBaseDist, "dist", eveArch, "installer", fmt.Sprintf("rootfs-%s.img", eveHV)), filepath.Join(eserverImageDist, "baseos", "baseos.qcow2")); err != nil {
			log.Errorf("Copy EVE base image failed: %s", err)
		} else {
			log.Info("Copy EVE base image done")
		}
		containerImageFile := filepath.Join(eserverImageDist, "docker", "alpine.tar")
		if _, err := os.Stat(containerImageFile); os.IsNotExist(err) {
			if err = utils.BuildContainer(dockerYML, defaultImageTag); err != nil {
				log.Errorf("Cannot build container image: %s", err)
			} else {
				log.Info("Container image build done")
			}
			if err = utils.DockerImageRepack(command, containerImageFile, defaultImageTag); err != nil {
				log.Errorf("Cannot repack container image: %s", err)
			} else {
				log.Info("Container image repack done")
			}
		} else {
			log.Info("Container image build done")
		}
		if _, err := os.Lstat(binDir); os.IsNotExist(err) {
			if err := os.MkdirAll(binDir, 0755); err != nil {
				log.Errorf("Cannot create binDir: %s", err)
			}
		}
		linuxKitPath := filepath.Join(binDir, fmt.Sprintf("linuxkit-%s-%s", runtime.GOOS, runtime.GOARCH))
		linuxKitSymlinkPath := filepath.Join(binDir, "linuxkit")
		if _, err := os.Stat(linuxKitPath); os.IsNotExist(err) {
			linuxKitUrl := fmt.Sprintf("https://github.com/linuxkit/linuxkit/releases/download/%s/linuxkit-%s-%s", defaultLinuxKitVersion, runtime.GOOS, runtime.GOARCH)
			if err = utils.DownloadFile(linuxKitPath, linuxKitUrl); err != nil {
				log.Errorf("Download LinuxKit from %s failed: %s", linuxKitUrl, err)
			} else {
				if err := os.Chmod(linuxKitPath, 755); err != nil {
					log.Errorf("Cannot Chmod LinuxKit: %s", err)
				}
				if err := os.Symlink(linuxKitPath, linuxKitSymlinkPath); err != nil {
					log.Errorf("Cannot make LinuxKit symlink: %s", err)
				}
			}
			log.Info("LinuxKit download done")
		} else {
			log.Info("LinuxKit download done")
		}
		vmImageFile := filepath.Join(eserverImageDist, "vm", "alpine.qcow2")
		if _, err := os.Stat(vmImageFile); os.IsNotExist(err) {
			if err = utils.BuildVM(linuxKitSymlinkPath, vmYML, vmImageFile); err != nil {
				log.Errorf("Cannot build VM image: %s", err)
			} else {
				log.Info("VM image build done")
			}
		} else {
			log.Info("VM image build done")
		}
		if _, err := os.Stat(qemuFileToSave); os.IsNotExist(err) {
			if err = utils.PrepareQEMUConfig(command, qemuFileToSave, qemuFirmware, qemuConfigPath, qemuDTBPath, translateHostFwd(qemuHostFwd)); err != nil {
				log.Errorf("Cannot prepare QEMU config: %s", err)
			} else {
				log.Info("Prepare QEMU config done")
			}
		} else {
			log.Info("Prepare QEMU config done")
		}
	},
}

func translateHostFwd(inp map[string]string) string {
	result := ""
	for k, v := range inp {
		result = fmt.Sprintf("%s,%s=%s", result, k, v)
	}
	return strings.Trim(result, ",")
}

var configEdenCmd = &cobra.Command{
	Use:   "eden",
	Short: "generate config eden",
	Long:  `Generate config eden.`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		var err error
		if config == "" {
			config, err = utils.DefaultConfigPath()
			if err != nil {
				log.Fatalf("fail in DefaultConfigPath: %s", err)
			}
		}
		if _, err := os.Stat(config); !os.IsNotExist(err) {
			if force {
				if err := os.Remove(config); err != nil {
					log.Fatal(err)
				}
			} else {
				log.Fatalf("config already exists: %s", config)
			}
		}
		if _, err := utils.LoadConfigFile(config); err != nil {
			log.Fatalf("error reading config: %s", err)
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
	},
}

func configEdenInit() {
	configPath, err := utils.DefaultConfigPath()
	if err != nil {
		log.Fatal(err)
	}
	configEdenCmd.Flags().BoolVarP(&force, "force", "", false, "force overwrite config file")
	if err := viper.BindPFlags(configEdenCmd.Flags()); err != nil {
		log.Fatal(err)
	}
	configEdenCmd.Flags().StringVar(&config, "config", configPath, "path to config file")
}

func configInit() {
	currentPath, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	configEdenInit()
	configCmd.AddCommand(configEdenCmd)

	configCmd.Flags().StringVarP(&certsDir, "certs-dist", "o", filepath.Join(currentPath, "dist", "certs"), "directory with certs")
	configCmd.Flags().StringVarP(&certsDomain, "domain", "d", defaultDomain, "FQDN for certificates")
	configCmd.Flags().StringVarP(&certsIP, "ip", "i", defaultIP, "IP address to use")
	configCmd.Flags().StringVarP(&certsUUID, "uuid", "u", defaultUUID, "UUID to use for device")

	configCmd.Flags().StringVarP(&adamDist, "adam-dist", "", filepath.Join(currentPath, "dist", "adam"), "adam dist to start (required)")
	configCmd.Flags().StringVarP(&adamPort, "adam-port", "", "3333", "adam dist to start")

	configCmd.Flags().StringSliceVarP(&qemuFirmware, "eve-firmware", "", nil, "firmware path")
	configCmd.Flags().StringVarP(&qemuConfigPath, "config-part", "", "", "path for config drive")
	configCmd.Flags().StringVarP(&qemuDTBPath, "dtb-part", "", "", "path for device tree drive (for arm)")
	configCmd.Flags().StringVarP(&eveImageFile, "image-file", "", "", "path for image drive (required)")
	configCmd.Flags().StringVarP(&eveDist, "eve-dist", "", filepath.Join(currentPath, "dist", "eve"), "directory to save EVE")
	configCmd.Flags().StringVarP(&eveBaseDist, "eve-base-dist", "", filepath.Join(currentPath, "dist", "evebaseos"), "directory to save Base image of EVE")
	configCmd.Flags().StringVarP(&eveRepo, "eve-repo", "", defaultEveRepo, "EVE repo")
	configCmd.Flags().StringVarP(&eveTag, "eve-tag", "", defaultEveTag, "EVE tag")
	configCmd.Flags().StringVarP(&eveArch, "eve-arch", "", runtime.GOARCH, "EVE arch")
	configCmd.Flags().StringVarP(&eveBaseTag, "eve-base-tag", "", defaultBaseEveTag, "tag of base image of EVE")
	configCmd.Flags().StringToStringVarP(&qemuHostFwd, "eve-hostfwd", "", defaultQemuHostFwd, "port forward map")
	configCmd.Flags().StringVarP(&qemuFileToSave, "qemu-config", "", filepath.Join(currentPath, "dist", defaultQemuFileToSave), "file to save qemu config")
	configCmd.Flags().BoolVarP(&download, "download", "", true, "download EVE or build")
	configCmd.Flags().StringVarP(&eveHV, "hv", "", "kvm", "hv of rootfs to use")

	configCmd.Flags().StringVarP(&eserverImageDist, "image-dist", "", filepath.Join(currentPath, "dist", "images"), "image dist for eserver")
	configCmd.Flags().StringVarP(&binDir, "bin-dist", "", filepath.Join(currentPath, "dist", "bin"), "directory for binaries")
	configCmd.Flags().StringVarP(&dockerYML, "docker-yml", "", filepath.Join(currentPath, "images", "docker", "alpine", "alpine.yml"), "directory for binaries")
	configCmd.Flags().StringVarP(&vmYML, "vm-yml", "", filepath.Join(currentPath, "images", "vm", "alpine", "alpine.yml"), "directory for binaries")
}
