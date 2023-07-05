package patchwork

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"

	"github.com/lf-edge/eden/pkg/controller"
	"github.com/lf-edge/eden/pkg/controller/einfo"
	"github.com/lf-edge/eden/pkg/eve"
	"github.com/lf-edge/eve/api/go/info"
	uuid "github.com/satori/go.uuid"
)

func checkAppState(ctrl controller.Cloud, devUUID uuid.UUID, appName string, eveState *eve.State, expState string, timeout time.Duration) error {
	startTime := time.Now()

	// Waiting for 15 min maximum to get eclient-mount app in state running
	handleInfo := func(im *info.ZInfoMsg) bool {
		eveState.InfoCallback()(im)
		for _, s := range eveState.Applications() {
			if s.Name == appName {
				if s.EVEState == expState {
					return true
				}
			}
		}
		if time.Now().After(startTime.Add(timeout)) {
			log.Fatal("eclient-mount timeout")
		}
		return false
	}

	if err := ctrl.InfoChecker(devUUID, nil, handleInfo, einfo.InfoNew, 0); err != nil {
		return fmt.Errorf("eclient-mount RUNNING state InfoChecker: %w", err)
	}

	return nil
}

func withCapturingStdout(f func() error) ([]byte, error) {
	rescueStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := f()

	w.Close()
	out, _ := ioutil.ReadAll(r)
	os.Stdout = rescueStdout

	return out, err
}

func checkOutput(input string, shouldHave, shouldNotHave []string) error {
	for _, str := range shouldHave {
		if !strings.Contains(input, str) {
			return fmt.Errorf("Input does not contain %v", str)
		}
	}

	for _, str := range shouldNotHave {
		if strings.Contains(input, str) {
			return fmt.Errorf("Input contains %v", str)
		}
	}

	return nil
}
