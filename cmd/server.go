package cmd

import (
	"github.com/lf-edge/eden/pkg/defaults"
	log "github.com/sirupsen/logrus"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/spf13/cobra"
)

var (
	serverPort int
	serverDir  string
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "start a server",
	Long:  `Start a server.`,
	Run: func(cmd *cobra.Command, args []string) {
		http.Handle("/", http.FileServer(http.Dir(serverDir)))

		log.Infof("Serving %s on HTTP port: %d\n", serverDir, serverPort)
		log.Fatal(http.ListenAndServe(":"+strconv.Itoa(serverPort), nil))
	},
}

func serverInit() {
	currentPath, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	serverCmd.Flags().IntVarP(&serverPort, "port", "p", defaults.DefaultEserverPort, "port to serve on")
	serverCmd.Flags().StringVarP(&serverDir, "directory", "d", filepath.Join(currentPath, defaults.DefaultDist, defaults.DefaultImageDist), "location of static root for server with files")
}
