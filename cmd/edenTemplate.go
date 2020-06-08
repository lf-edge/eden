package cmd

import (
	"fmt"
	"io/ioutil"
	
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/lf-edge/eden/pkg/utils"
)

var templateCmd = &cobra.Command{
	Use:   "template <file>",
	Short: "Render template",
	Long: ``,
	Args: cobra.ExactArgs(1),
	PreRunE: func(cmd *cobra.Command, args []string) error {
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		tmpl, err := ioutil.ReadFile(args[0])
		if err != nil {
			log.Fatal(err)
		}
		
		out, err := utils.RenderTemplate(configFile, string(tmpl))
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(out)
	},
}

func templateInit() {
}
