package sec_test

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/openevec"
	"github.com/lf-edge/eden/pkg/utils"
)

type remoteNode struct {
	openEVEC *openevec.OpenEVEC
}

func getOpenEVEC() *openevec.OpenEVEC {
	edenConfigEnv := os.Getenv(defaults.DefaultConfigEnv)
	configName := utils.GetConfig(edenConfigEnv)

	viperCfg, err := openevec.FromViper(configName, "debug")
	if err != nil {
		return nil
	}

	return openevec.CreateOpenEVEC(viperCfg)
}

func createRemoteNode() *remoteNode {
	evec := getOpenEVEC()
	if evec == nil {
		return nil
	}

	return &remoteNode{openEVEC: evec}
}

func (node *remoteNode) runCommand(command string) ([]byte, error) {
	realStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		return nil, err
	}

	os.Stdout = w

	// unfortunately, we can't capture command return value from SSHEve
	err = node.openEVEC.SSHEve(command)

	os.Stdout = realStdout
	w.Close()

	if err != nil {
		return nil, err
	}

	out, _ := io.ReadAll(r)
	return out, nil
}

func (node *remoteNode) fileExists(fileName string) (bool, error) {
	command := fmt.Sprintf("if stat \"%s\"; then echo \"1\"; else echo \"0\"; fi", fileName)
	out, err := node.runCommand(command)
	if err != nil {
		return false, err
	}

	if strings.TrimSpace(string(out)) == "0" {
		return false, nil
	}

	return true, nil
}

func (node *remoteNode) readFile(fileName string) ([]byte, error) {
	exist, err := node.fileExists(fileName)
	if err != nil {
		return nil, err
	}

	if !exist {
		return nil, fmt.Errorf("file %s does not exist", fileName)
	}

	command := fmt.Sprintf("cat %s", fileName)
	return node.runCommand(command)
}
