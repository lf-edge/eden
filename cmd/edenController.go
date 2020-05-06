package cmd

import (
	"fmt"
	"github.com/lf-edge/eden/pkg/utils"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"regexp"
	"strings"
)

var controllerMode string

func getParams(line, regEx string) (paramsMap map[string]string) {

	var compRegEx = regexp.MustCompile(regEx)
	match := compRegEx.FindStringSubmatch(strings.TrimSpace(line))

	paramsMap = make(map[string]string)
	for i, name := range compRegEx.SubexpNames() {
		if i > 0 && i <= len(match) {
			paramsMap[name] = match[i]
		}
	}
	return
}

func getControllerMode() (modeType, modeURL string, err error) {
	params := getParams(controllerMode, controllerModePattern)
	if len(params) == 0 {
		return "", "", fmt.Errorf("cannot parse mode (not [file|proto|adam|zedcloud]://<URL>): %s", controllerMode)
	}
	ok := false
	if modeType, ok = params["Type"]; !ok {
		return "", "", fmt.Errorf("cannot parse modeType (not [file|proto|adam|zedcloud]://<URL>): %s", controllerMode)
	}
	if modeURL, ok = params["URL"]; !ok {
		return "", "", fmt.Errorf("cannot parse modeURL (not [file|proto|adam|zedcloud]://<URL>): %s", controllerMode)
	}
	return
}

var controllerCmd = &cobra.Command{
	Use:   "controller",
	Short: "interact with controller",
	Long:  `Interact with controller.`,
}

var edgeNode = &cobra.Command{
	Use:   "edge-node",
	Short: "manage EVE instance",
	Long:  `Manage EVE instance.`,
}

var edgeNodeReboot = &cobra.Command{
	Use:   "reboot",
	Short: "reboot EVE instance",
	Long:  `reboot EVE instance.`,
	Run: func(cmd *cobra.Command, args []string) {
		modeType, modeURL, err := getControllerMode()
		if err != nil {
			log.Fatal(err)
		}
		log.Infof("Mode type: %s", modeType)
		log.Infof("Mode url: %s", modeURL)
		var changer configChanger
		switch modeType {
		case "file":
			changer = &fileChanger{fileConfig: modeURL}
		case "adam":
			changer = &adamChanger{adamUrl: modeURL}

		default:
			log.Fatalf("Not implemented type: %s", modeType)
		}

		ctrl, dev, err := changer.getControllerAndDev()
		if err != nil {
			log.Fatalf("getControllerAndDev error: %s", err)
		}
		rebootCounter, _ := dev.GetRebootCounter()
		dev.SetRebootCounter(rebootCounter+1, true)
		if err = changer.setControllerAndDev(ctrl, dev); err != nil {
			log.Fatalf("setControllerAndDev error: %s", err)
		}
		log.Info("Reboot request has been sent")
	},
}

func controllerInit() {
	configPath, err := utils.DefaultConfigPath()
	if err != nil {
		log.Fatal(err)
	}
	controllerCmd.AddCommand(edgeNode)
	edgeNode.AddCommand(edgeNodeReboot)
	pf := controllerCmd.PersistentFlags()
	pf.StringVarP(&controllerMode, "mode", "m", "", "mode to use [file|proto|adam|zedcloud]://<URL>")
	pf.StringVar(&configFile, "config", configPath, "path to config file")
	if err = cobra.MarkFlagRequired(pf, "mode"); err != nil {
		log.Fatal(err)
	}
}
