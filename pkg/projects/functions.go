package projects

import (
	"fmt"
	"strings"

	"github.com/lf-edge/eden/pkg/controller/eapps"
	"github.com/lf-edge/eden/pkg/device"
	"github.com/lf-edge/eve/api/go/logs"
	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
)

//CheckMessageInAppLog try to find message in logs of app
func (tc *TestContext) CheckMessageInAppLog(edgeNode *device.Ctx, appID uuid.UUID, message string) ProcTimerFunc {
	return func() error {
		foundedMessage := ""
		handler := func(le *logs.LogEntry) bool {
			if strings.Contains(le.Content, message) {
				foundedMessage = le.Content
				return true
			}
			return false
		}
		if err := tc.GetController().LogAppsChecker(edgeNode.GetID(), appID, nil, handler, eapps.LogExist, 0); err != nil {
			log.Fatalf("LogAppsChecker: %s", err)
		}
		if foundedMessage != "" {
			return fmt.Errorf("founded in app logs: %s", foundedMessage)
		}
		return nil
	}
}
