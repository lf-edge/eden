package projects

import (
	"fmt"
	"strings"

	"github.com/lf-edge/eden/pkg/controller/eapps"
	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/device"
	"github.com/lf-edge/eve/api/go/logs"
	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
)

//CheckMessageInAppLog try to find message in logs of app
func (tc *TestContext) CheckMessageInAppLog(edgeNode *device.Ctx, appID uuid.UUID, message string, callbacks ...Callback) ProcTimerFunc {
	return func() error {
		foundedMessage := ""
		handler := func(le *logs.LogEntry) bool {
			if strings.Contains(le.Content, message) {
				foundedMessage = strings.TrimSpace(le.Content)
				return true
			}
			return false
		}
		if err := tc.GetController().LogAppsChecker(edgeNode.GetID(), appID, nil, handler, eapps.LogExist, 0); err != nil {
			log.Fatalf("LogAppsChecker: %s", err)
		}
		if foundedMessage != "" {
			for _, clb := range callbacks {
				clb()
			}
			return fmt.Errorf("founded in app logs: %s", foundedMessage)
		}
		return nil
	}
}

//SendCommandSSH try to access SSH with timer and sends command
func SendCommandSSH(ip *string, port *int, user, password, command string, foreground bool, callbacks ...Callback) ProcTimerFunc {
	return func() error {
		if *ip != "" {
			configSSH := &ssh.ClientConfig{
				User: user,
				Auth: []ssh.AuthMethod{
					ssh.Password(password),
				},
				HostKeyCallback: ssh.InsecureIgnoreHostKey(),
				Timeout:         defaults.DefaultRepeatTimeout,
			}
			conn, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", *ip, *port), configSSH)
			if err != nil {
				return nil
			}
			session, _ := conn.NewSession()
			if foreground {
				defer session.Close()
				if err := session.Run(command); err != nil {
					fmt.Println(err)
					return nil
				}
			} else {
				go func() {
					_ = session.Run(command) //we cannot get answer for this command
					session.Close()
				}()
			}
			for _, clb := range callbacks {
				clb()
			}
			return fmt.Errorf("command \"%s\" sended via SSH on %s:%d", command, *ip, *port)
		}
		return nil
	}
}
