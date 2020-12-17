package cmd

import (
	"fmt"

	"github.com/lf-edge/eden/pkg/utils"
	"github.com/lf-edge/eve/api/go/config"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var volumeCmd = &cobra.Command{
	Use: "volume",
}

type volInstState struct {
	name          string
	uuid          string
	image         string
	volType       config.Format
	size          string
	maxSize       string
	contentTreeID string
	adamState     string
	eveState      string
	deleted       bool
}

func volInstStateHeader() string {
	return "NAME\tUUID\tIMAGE\tTYPE\tSIZE\tMAX_SIZE\tSTATE(ADAM)\tLAST_STATE(EVE)"
}

func (volInstStateObj *volInstState) toString() string {
	return fmt.Sprintf("%s\t%s\t%s\t%v\t%s\t%s\t%s\t%s",
		volInstStateObj.name, volInstStateObj.uuid, volInstStateObj.image,
		volInstStateObj.volType, volInstStateObj.size, volInstStateObj.maxSize,
		volInstStateObj.adamState, volInstStateObj.eveState)
}

//networkLsCmd is a command to list deployed volumes
var volumeLsCmd = &cobra.Command{
	Use:   "ls",
	Short: "List volumes",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		assignCobraToViper(cmd)
		_, err := utils.LoadConfigFile(configFile)
		if err != nil {
			return fmt.Errorf("error reading config: %s", err.Error())
		}
		devModel = viper.GetString("eve.devmodel")
		qemuPorts = viper.GetStringMapString("eve.hostfwd")
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		if err := volumeList(log.GetLevel()); err != nil {
			log.Fatal(err)
		}
	},
}

func volumeInit() {
	volumeCmd.AddCommand(volumeLsCmd)
}
