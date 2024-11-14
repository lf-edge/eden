package testcontext

import (
	"fmt"
	"strings"

	"github.com/lf-edge/eden/pkg/controller/eapps"
	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/device"
	"github.com/lf-edge/eve-api/go/logs"
	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
	"github.com/tmc/scp"
	"golang.org/x/crypto/ssh"
)

// CheckMessageInAppLog try to find message in logs of app
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

// SendCommandSSH try to access SSH with timer and sends command
func SendCommandSSH(ip *string, port *int, user, password, command string, foreground bool, callbacks ...Callback) ProcTimerFunc {
	return func() error {
		if ip != nil && *ip != "" {
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
				fmt.Printf("No ssh connections: %v", err)
				return nil
			}
			session, err := conn.NewSession()
			if err != nil {
				fmt.Printf("Error creating session: %v", err)
				return nil
			}
			if foreground {
				defer conn.Close()
				defer session.Close()
				if err := session.Run(command); err != nil {
					fmt.Println(err)
					return nil
				}
			} else {
				go func() {
					_ = session.Run(command) //we cannot get answer for this command
					session.Close()
					conn.Close()
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

// SendFileSCP sends a file over SCP
func SendFileSCP(ip *string, port *int, user, password, filename, destpath string) ProcTimerFunc {
	return func() error {
		if ip != nil && *ip != "" {
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
				fmt.Printf("No ssh connections: %v", err)
				return nil
			}

			session, err := conn.NewSession()
			if err != nil {
				fmt.Printf("Create new session failed: %v\n", err)
				return nil
			}

			err = scp.CopyPath(filename, destpath, session)
			if err != nil {
				fmt.Printf("Copy file on guest VM failed: %v\n", err)
				return nil
			}
			return fmt.Errorf("scp of file %s done", filename)
		}
		return nil
	}
}
