package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/openevec"
	"github.com/lf-edge/eden/pkg/utils"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func newDebugCmd(cfg *openevec.EdenSetupArgs) *cobra.Command {
	var debugCmd = &cobra.Command{
		Use: "debug",
	}

	/*
		if viperLoaded {
			eveSSHKey = utils.ResolveAbsPath(viper.GetString("eden.ssh-key"))
			extension := filepath.Ext(eveSSHKey)
			eveSSHKey = strings.TrimRight(eveSSHKey, extension)
			eveRemote = viper.GetBool("eve.remote")
			eveRemoteAddr = viper.GetString("eve.remote-addr")
			if eveRemote || eveRemoteAddr == "" {
				if !cmd.Flags().Changed("eve-ssh-port") {
					eveSSHPort = 22
				}
			}
		}
	*/

	groups := CommandGroups{
		{
			Message: "Basic Commands",
			Commands: []*cobra.Command{
				newDebugStartEveCmd(cfg),
				newDebugStopEveCmd(cfg),
				newDebugSaveEveCmd(cfg),
				newDebugHardwareEveCmd(cfg),
			},
		},
	}

	groups.AddTo(debugCmd)

	return debugCmd
}

func newDebugStartEveCmd(cfg *openevec.EdenSetupArgs) *cobra.Command {
	var eveSSHKey, eveHost, perfOptions, perfLocation string
	var eveSSHPort int

	var debugStartEveCmd = &cobra.Command{
		Use:   "start",
		Short: "start perf in EVE",
		Run: func(cmd *cobra.Command, args []string) {
			if _, err := os.Stat(cfg.Eden.SSHKey); !os.IsNotExist(err) {
				commandToRun := fmt.Sprintf("perf record %s -o %s", perfOptions, perfLocation)
				commandToRun = fmt.Sprintf("sh -c 'nohup %s > /dev/null 2>&1 &'", commandToRun)
				if err = openevec.SdnForwardSSHToEve(commandToRun, cfg); err != nil {
					log.Fatal(err)
				}
			} else {
				log.Fatalf("SSH key problem: %s", err)
			}
		},
	}

	currentPath, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	debugStartEveCmd.Flags().StringVarP(&eveSSHKey, "ssh-key", "", filepath.Join(currentPath, defaults.DefaultCertsDist, "id_rsa"), "file to use for ssh access")
	debugStartEveCmd.Flags().StringVarP(&eveHost, "eve-host", "", defaults.DefaultEVEHost, "IP of eve")
	debugStartEveCmd.Flags().IntVarP(&eveSSHPort, "eve-ssh-port", "", defaults.DefaultSSHPort, "Port for ssh access")
	debugStartEveCmd.Flags().StringVar(&perfOptions, "perf-options", "-F 99 -a -g", "Options for perf record")
	debugStartEveCmd.Flags().StringVar(&perfLocation, "perf-location", defaults.DefaultPerfEVELocation, "Perf output location on EVE")

	addSdnPortOpts(debugStartEveCmd, cfg)

	return debugStartEveCmd
}

func newDebugStopEveCmd(cfg *openevec.EdenSetupArgs) *cobra.Command {
	var eveSSHKey, eveHost string
	var eveSSHPort int
	var debugStopEveCmd = &cobra.Command{
		Use:   "stop",
		Short: "stop perf in EVE",
		Run: func(cmd *cobra.Command, args []string) {
			if _, err := os.Stat(eveSSHKey); !os.IsNotExist(err) {
				commandToRun := "killall -SIGINT perf"
				if err = openevec.SdnForwardSSHToEve(commandToRun, cfg); err != nil {
					log.Fatal(err)
				}
			} else {
				log.Fatalf("SSH key problem: %s", err)
			}
		},
	}

	currentPath, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	debugStopEveCmd.Flags().StringVarP(&eveSSHKey, "ssh-key", "", filepath.Join(currentPath, defaults.DefaultCertsDist, "id_rsa"), "file to use for ssh access")
	debugStopEveCmd.Flags().StringVarP(&eveHost, "eve-host", "", defaults.DefaultEVEHost, "IP of eve")
	debugStopEveCmd.Flags().IntVarP(&eveSSHPort, "eve-ssh-port", "", defaults.DefaultSSHPort, "Port for ssh access")

	addSdnPortOpts(debugStopEveCmd, cfg)

	return debugStopEveCmd
}

func newDebugSaveEveCmd(cfg *openevec.EdenSetupArgs) *cobra.Command {
	var eveSSHKey, eveHost, perfLocation string
	var eveSSHPort int

	var debugSaveEveCmd = &cobra.Command{
		Use:   "save <file>",
		Short: "save file with perf script output from EVE, create svg and save to provided file",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			absPath, err := filepath.Abs(args[0])
			if err != nil {
				log.Fatal(err)
			}
			tmpFile := fmt.Sprintf("%s.tmp", absPath)
			if _, err := os.Stat(eveSSHKey); !os.IsNotExist(err) {
				commandToRun := fmt.Sprintf("perf script -i %s > %s", perfLocation, defaults.DefaultPerfScriptEVELocation)
				if err = openevec.SdnForwardSSHToEve(commandToRun, cfg); err != nil {
					log.Fatal(err)
				}
				err = openevec.SdnForwardSCPFromEve(defaults.DefaultPerfScriptEVELocation, tmpFile, cfg)
				if err != nil {
					log.Fatal(err)
				}
				image := fmt.Sprintf("%s:%s", defaults.DefaultProcContainerRef, defaults.DefaultProcTag)
				commandToRun = fmt.Sprintf("-i /in/%s -o /out/%s svg", filepath.Base(tmpFile), filepath.Base(absPath))
				volumeMap := map[string]string{"/in": filepath.Dir(tmpFile), "/out": filepath.Dir(absPath)}
				var result string
				if result, err = utils.RunDockerCommand(image, commandToRun, volumeMap); err != nil {
					log.Fatal(err)
				}
				fmt.Println(result)
				log.Infof("Please see output inside %s", absPath)
			} else {
				log.Fatalf("SSH key problem: %s", err)
			}
		},
	}

	currentPath, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	debugSaveEveCmd.Flags().StringVarP(&eveSSHKey, "ssh-key", "", filepath.Join(currentPath, defaults.DefaultCertsDist, "id_rsa"), "file to use for ssh access")
	debugSaveEveCmd.Flags().StringVarP(&eveHost, "eve-host", "", defaults.DefaultEVEHost, "IP of eve")
	debugSaveEveCmd.Flags().IntVarP(&eveSSHPort, "eve-ssh-port", "", defaults.DefaultSSHPort, "Port for ssh access")
	debugSaveEveCmd.Flags().StringVar(&perfLocation, "perf-location", defaults.DefaultPerfEVELocation, "Perf output location on EVE")

	addSdnPortOpts(debugSaveEveCmd, cfg)

	return debugSaveEveCmd
}

func newDebugHardwareEveCmd(cfg *openevec.EdenSetupArgs) *cobra.Command {
	var eveSSHKey, eveHost, hwLocation string
	var eveSSHPort int
	var short bool

	var debugHardwareEveCmd = &cobra.Command{
		Use:   "hw <file>",
		Short: "save file with lshw output from EVE",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			absPath, err := filepath.Abs(args[0])
			if err != nil {
				log.Fatal(err)
			}
			if _, err := os.Stat(eveSSHKey); !os.IsNotExist(err) {
				commandToRun := "lshw"
				if short {
					commandToRun = "lshw -short"
				}
				commandToRun += ">" + hwLocation
				if err = openevec.SdnForwardSSHToEve(commandToRun, cfg); err != nil {
					log.Fatal(err)
				}
				if err = openevec.SdnForwardSCPFromEve(hwLocation, absPath, cfg); err != nil {
					log.Fatal(err)
				}
			}
		},
	}

	currentPath, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	debugHardwareEveCmd.Flags().StringVarP(&eveSSHKey, "ssh-key", "", filepath.Join(currentPath, defaults.DefaultCertsDist, "id_rsa"), "file to use for ssh access")
	debugHardwareEveCmd.Flags().StringVarP(&eveHost, "eve-host", "", defaults.DefaultEVEHost, "IP of eve")
	debugHardwareEveCmd.Flags().IntVarP(&eveSSHPort, "eve-ssh-port", "", defaults.DefaultSSHPort, "Port for ssh access")
	debugHardwareEveCmd.Flags().StringVar(&hwLocation, "hw-location", defaults.DefaultHWEVELocation, "Hardware output location on EVE")
	debugHardwareEveCmd.Flags().BoolVar(&short, "short", true, "Short hardware info")

	addSdnPortOpts(debugHardwareEveCmd, cfg)

	return debugHardwareEveCmd
}
