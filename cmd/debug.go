package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/utils"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var debugCmd = &cobra.Command{
	Use: "debug",
}

var debugStartEveCmd = &cobra.Command{
	Use:   "start",
	Short: "start perf in EVE",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		assignCobraToViper(cmd)
		viperLoaded, err := utils.LoadConfigFile(configFile)
		if err != nil {
			return fmt.Errorf("error reading config: %s", err.Error())
		}
		if viperLoaded {
			eveSSHKey = utils.ResolveAbsPath(viper.GetString("eden.ssh-key"))
			extension := filepath.Ext(eveSSHKey)
			eveSSHKey = strings.TrimRight(eveSSHKey, extension)
			eveRemote = viper.GetBool("eve.remote")
			eveRemoteAddr = viper.GetString("eve.remote-addr")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		eveIP := getEVEIP()
		if eveIP == "" {
			log.Fatal("Np EVE IP")
		}
		if eveRemote || eveRemoteAddr == "" {
			if !cmd.Flags().Changed("eve-ssh-port") {
				eveSSHPort = 22
			}
		}
		if _, err := os.Stat(eveSSHKey); !os.IsNotExist(err) {
			commandToRun := fmt.Sprintf("perf record -F 99 -a -g -o %s", defaults.DefaultPerfEVELocation)
			arguments := fmt.Sprintf("-o ConnectTimeout=5 -oStrictHostKeyChecking=no -i %s -p %d root@%s %s", eveSSHKey, eveSSHPort, eveIP, commandToRun)
			log.Debugf("Try to ssh %s:%d with key %s and command %s", eveHost, eveSSHPort, eveSSHKey, arguments)
			if _, err := utils.RunCommandBackground("ssh", nil, strings.Fields(arguments)...); err != nil {
				log.Fatalf("ssh error for command %s: %s", commandToRun, err)
			}
		} else {
			log.Fatalf("SSH key problem: %s", err)
		}
	},
}

var debugStopEveCmd = &cobra.Command{
	Use:   "stop",
	Short: "stop perf in EVE",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		assignCobraToViper(cmd)
		viperLoaded, err := utils.LoadConfigFile(configFile)
		if err != nil {
			return fmt.Errorf("error reading config: %s", err.Error())
		}
		if viperLoaded {
			eveSSHKey = utils.ResolveAbsPath(viper.GetString("eden.ssh-key"))
			extension := filepath.Ext(eveSSHKey)
			eveSSHKey = strings.TrimRight(eveSSHKey, extension)
			eveRemote = viper.GetBool("eve.remote")
			eveRemoteAddr = viper.GetString("eve.remote-addr")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		eveIP := getEVEIP()
		if eveIP == "" {
			log.Fatal("Np EVE IP")
		}
		if eveRemote || eveRemoteAddr == "" {
			if !cmd.Flags().Changed("eve-ssh-port") {
				eveSSHPort = 22
			}
		}
		if _, err := os.Stat(eveSSHKey); !os.IsNotExist(err) {
			commandToRun := "killall perf"
			arguments := fmt.Sprintf("-o ConnectTimeout=5 -oStrictHostKeyChecking=no -i %s -p %d root@%s %s", eveSSHKey, eveSSHPort, eveIP, commandToRun)
			log.Debugf("Try to ssh %s:%d with key %s and command %s", eveHost, eveSSHPort, eveSSHKey, arguments)
			if err := utils.RunCommandForeground("ssh", strings.Fields(arguments)...); err != nil {
				log.Fatalf("ssh error for command %s: %s", commandToRun, err)
			}
		} else {
			log.Fatalf("SSH key problem: %s", err)
		}
	},
}

var debugSaveEveCmd = &cobra.Command{
	Use:   "save <file>",
	Short: "save file with perf script output from EVE, create svg and save to provided file",
	Args:  cobra.ExactArgs(1),
	PreRunE: func(cmd *cobra.Command, args []string) error {
		assignCobraToViper(cmd)
		viperLoaded, err := utils.LoadConfigFile(configFile)
		if err != nil {
			return fmt.Errorf("error reading config: %s", err.Error())
		}
		if viperLoaded {
			eveSSHKey = utils.ResolveAbsPath(viper.GetString("eden.ssh-key"))
			extension := filepath.Ext(eveSSHKey)
			eveSSHKey = strings.TrimRight(eveSSHKey, extension)
			eveRemote = viper.GetBool("eve.remote")
			eveRemoteAddr = viper.GetString("eve.remote-addr")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		absPath, err := filepath.Abs(args[0])
		if err != nil {
			log.Fatal(err)
		}
		tmpFile := fmt.Sprintf("%s.tmp", absPath)
		eveIP := getEVEIP()
		if eveIP == "" {
			log.Fatal("Np EVE IP")
		}
		if eveRemote || eveRemoteAddr == "" {
			if !cmd.Flags().Changed("eve-ssh-port") {
				eveSSHPort = 22
			}
		}
		if _, err := os.Stat(eveSSHKey); !os.IsNotExist(err) {
			commandToRun := fmt.Sprintf("perf script -i %s > %s", defaults.DefaultPerfEVELocation, defaults.DefaultPerfScriptEVELocation)
			arguments := fmt.Sprintf("-o ConnectTimeout=5 -oStrictHostKeyChecking=no -i %s -p %d root@%s %s", eveSSHKey, eveSSHPort, eveIP, commandToRun)
			log.Debugf("Try to ssh %s:%d with key %s and command %s", eveHost, eveSSHPort, eveSSHKey, arguments)
			if err := utils.RunCommandForeground("ssh", strings.Fields(arguments)...); err != nil {
				log.Fatalf("ssh error for command %s: %s", commandToRun, err)
			}
			commandToRun = fmt.Sprintf("%s %s", defaults.DefaultPerfScriptEVELocation, tmpFile)
			arguments = fmt.Sprintf("-o ConnectTimeout=5 -oStrictHostKeyChecking=no -i %s -P %d root@%s:%s", eveSSHKey, eveSSHPort, eveIP, commandToRun)
			log.Debugf("Try to scp %s:%d with key %s and command %s", eveHost, eveSSHPort, eveSSHKey, arguments)
			if err := utils.RunCommandForeground("scp", strings.Fields(arguments)...); err != nil {
				log.Fatalf("scp error for command %s: %s", commandToRun, err)
			}
			image := fmt.Sprintf("%s:%s", defaults.DefaultProcContainerRef, defaults.DefaultProcTag)
			commandToRun = fmt.Sprintf("-i /in/%s -o /out/%s svg", filepath.Base(tmpFile), filepath.Base(absPath))
			volumeMap := map[string]string{"/in": filepath.Dir(tmpFile), "/out": filepath.Dir(absPath)}
			if _, err := utils.RunDockerCommand(image, commandToRun, volumeMap); err != nil {
				log.Fatal(err)
			}
			log.Infof("Please see output inside %s", absPath)
		} else {
			log.Fatalf("SSH key problem: %s", err)
		}
	},
}

func debugInit() {
	currentPath, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	debugCmd.AddCommand(debugStartEveCmd)
	debugCmd.AddCommand(debugStopEveCmd)
	debugCmd.AddCommand(debugSaveEveCmd)
	debugStartEveCmd.Flags().StringVarP(&eveSSHKey, "ssh-key", "", filepath.Join(currentPath, defaults.DefaultCertsDist, "id_rsa"), "file to use for ssh access")
	debugStartEveCmd.Flags().StringVarP(&eveHost, "eve-host", "", defaults.DefaultEVEHost, "IP of eve")
	debugStartEveCmd.Flags().IntVarP(&eveSSHPort, "eve-ssh-port", "", defaults.DefaultSSHPort, "Port for ssh access")
	debugStopEveCmd.Flags().StringVarP(&eveSSHKey, "ssh-key", "", filepath.Join(currentPath, defaults.DefaultCertsDist, "id_rsa"), "file to use for ssh access")
	debugStopEveCmd.Flags().StringVarP(&eveHost, "eve-host", "", defaults.DefaultEVEHost, "IP of eve")
	debugStopEveCmd.Flags().IntVarP(&eveSSHPort, "eve-ssh-port", "", defaults.DefaultSSHPort, "Port for ssh access")
	debugSaveEveCmd.Flags().StringVarP(&eveSSHKey, "ssh-key", "", filepath.Join(currentPath, defaults.DefaultCertsDist, "id_rsa"), "file to use for ssh access")
	debugSaveEveCmd.Flags().StringVarP(&eveHost, "eve-host", "", defaults.DefaultEVEHost, "IP of eve")
	debugSaveEveCmd.Flags().IntVarP(&eveSSHPort, "eve-ssh-port", "", defaults.DefaultSSHPort, "Port for ssh access")
}
