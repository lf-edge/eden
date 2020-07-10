package cmd

import (
	"github.com/lf-edge/eden/eserver/pkg/manager"
	"github.com/lf-edge/eden/eserver/pkg/server"
	"github.com/spf13/cobra"
)

const (
	defaultPort = "8888"
	defaultIP   = "0.0.0.0"
)

var (
	port      string
	hostIP    string
	serverDir string
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "start a server",
	Long:  `Start a server.`,
	Run: func(cmd *cobra.Command, args []string) {
		server := &server.EServer{
			Port:    port,
			Address: hostIP,
			Manager: &manager.EServerManager{Dir: serverDir},
		}
		server.Start()
	},
}

func serverInit() {
	serverCmd.Flags().StringVar(&port, "port", defaultPort, "port on which to listen")
	serverCmd.Flags().StringVar(&hostIP, "ip", defaultIP, "IP address on which to listen")
	serverCmd.Flags().StringVar(&serverDir, "dir", "./run/eserver", "location of files to save")
}
