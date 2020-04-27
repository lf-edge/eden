package cmd

import (
	log "github.com/sirupsen/logrus"
	"net/http"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var (
	serverPort string
	serverDir  string
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "start a server",
	Long:  `Start a server.`,
	Run: func(cmd *cobra.Command, args []string) {
		http.Handle("/", http.FileServer(http.Dir(serverDir)))

		log.Infof("Serving %s on HTTP port: %s\n", serverDir, serverPort)
		log.Fatal(http.ListenAndServe(":"+serverPort, nil))
	},
}

func serverInit() {
	currentPath, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	serverCmd.Flags().StringVarP(&serverPort, "port", "p", defaultEserverPort, "port to serve on")
	serverCmd.Flags().StringVarP(&serverDir, "directory", "d", filepath.Join(currentPath, "dist", "images"), "location of static root for server with files")
}
