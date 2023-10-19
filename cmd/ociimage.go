package cmd

import (
	"github.com/lf-edge/eden/pkg/defaults"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func newOciImageCmd() *cobra.Command {
	var (
		fileToSave string
		image      string
		registry   string
		isLocal    bool
	)

	var ociImageCmd = &cobra.Command{
		Use:   "ociimage",
		Short: "do oci image manipulations",
		Long:  `Do oci image manipulations.`,
		Run: func(cmd *cobra.Command, args []string) {
			if err := openEVEC.OciImage(fileToSave, image, registry, isLocal); err != nil {
				log.Fatal(err)
			}
		},
	}

	ociImageCmd.Flags().StringVarP(&fileToSave, "output", "o", defaults.DefaultFileToSave, "file to save")
	ociImageCmd.Flags().StringVarP(&image, "image", "i", defaults.DefaultImage, "image to save")
	ociImageCmd.Flags().StringVarP(&registry, "registry", "r", defaults.DefaultRegistry, "registry")
	ociImageCmd.Flags().BoolVarP(&isLocal, "local", "l", defaults.DefaultIsLocal, "use local docker image")

	return ociImageCmd
}
