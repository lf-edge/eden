package cmd

import (
	"github.com/lf-edge/eden/pkg/utils"
	"github.com/spf13/cobra"
	"log"
	"os"
	"path"
	"path/filepath"
	"runtime"
)

var (
	adamDist         string
	adamPort         string
	adamForce        bool
	eserverImageDist string
	eserverPort      string
	eserverPidFile   string
	eserverLogFile   string
	evePidFile       string
	eveLogFile       string
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "start harness",
	Long:  `Start harness.`,
	Run: func(cmd *cobra.Command, args []string) {
		adamPath, err := filepath.Abs(adamDist)
		if err != nil {
			log.Fatalf("adam-dist problems: %s", err)
		}
		command, err := os.Executable()
		if err != nil {
			log.Fatalf("cannot obtain executable path: %s", err)
		}
		log.Printf("Executable path: %s", command)
		err = utils.StartAdam(adamPort, adamPath, adamForce)
		if err != nil {
			log.Fatalf("cannot start adam: %s", err)
		}
		log.Printf("Adam is running and accesible on port %s", adamPort)
		err = utils.StartEServer(command, eserverPort, eserverImageDist, eserverLogFile, eserverPidFile)
		if err != nil {
			log.Fatalf("cannot start eserver: %s", err)
		}
		log.Printf("Eserver is running and accesible on port %s", eserverPort)
		err = utils.StartEVEQemu(command, qemuARCH, qemuOS, qemuSMBIOSSerial, qemuAccel, qemuConfigFile, eveLogFile, evePidFile)
		if err != nil {
			log.Fatalf("cannot start eve: %s", err)
		}
		log.Printf("EVE is running")
	},
}

func startInit() {
	currentPath, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	startCmd.Flags().StringVarP(&adamDist, "adam-dist", "", path.Join(currentPath, "dist", "adam"), "adam dist to start (required)")
	startCmd.Flags().StringVarP(&adamPort, "adam-port", "", "3333", "adam dist to start")
	startCmd.Flags().BoolVarP(&adamForce, "adam-force", "", false, "adam force rebuild")
	startCmd.Flags().StringVarP(&eserverImageDist, "image-dist", "", path.Join(currentPath, "dist", "images"), "image dist for eserver")
	startCmd.Flags().StringVarP(&eserverPort, "eserver-port", "", "8888", "eserver port")
	startCmd.Flags().StringVarP(&eserverPidFile, "eserver-pid", "", path.Join(currentPath, "dist", "eserver.pid"), "file for save eserver pid")
	startCmd.Flags().StringVarP(&eserverLogFile, "eserver-log", "", path.Join(currentPath, "dist", "eserver.log"), "file for save eserver log")
	startCmd.Flags().StringVarP(&qemuARCH, "eve-arch", "", runtime.GOARCH, "arch of system")
	startCmd.Flags().StringVarP(&qemuOS, "eve-os", "", runtime.GOOS, "os to run on")
	startCmd.Flags().BoolVarP(&qemuAccel, "eve-accel", "", true, "use acceleration")
	startCmd.Flags().StringVarP(&qemuSMBIOSSerial, "eve-serial", "", "", "SMBIOS serial")
	startCmd.Flags().StringVarP(&qemuConfigFile, "eve-config", "", path.Join(currentPath, "dist", "qemu.conf"), "config file to use")
	startCmd.Flags().StringVarP(&evePidFile, "eve-pid", "", path.Join(currentPath, "dist", "eve.pid"), "file for save EVE pid")
	startCmd.Flags().StringVarP(&eveLogFile, "eve-log", "", path.Join(currentPath, "dist", "eve.log"), "file for save EVE log")
}
