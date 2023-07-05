package patchwork

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/lf-edge/eden/pkg/controller"
	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/eve"
	ec "github.com/lf-edge/eden/pkg/openevec"
)

func TestEclientMount(t *testing.T) {
	ctrl, err := controller.CloudPrepare()
	if err != nil {
		t.Errorf("CloudPrepare: %v", err)
		return
	}
	devFirst, err := ctrl.GetDeviceCurrent()
	if err != nil {
		t.Errorf("GetDeviceCurrent error: %v", err)
		return
	}
	devUUID := devFirst.GetID()

	port := 2223
	curPath, err := os.Getwd()
	if err != nil {
		t.Errorf("Getwd error: %v", err)
	}
	cfg := ec.GetDefaultConfig(curPath)
	pc := ec.GetDefaultPodConfig()
	pc.PortPublish = []string{fmt.Sprint(port)}
	podName := "eclient-mount"
	pc.Mount = []string{
		"src=docker://hello-world:linux,dst=/tst",
		fmt.Sprintf("src=%s/eclient/testdata,dst=/dir", cfg.Eden.TestScenario),
	}
	appLink := fmt.Sprintf("docker://%s:%s", defaults.DefaultEClientTag, defaults.DefaultEClientContainerRef)

	if err = ec.PodDeploy(appLink, *pc, cfg); err != nil {
		t.Errorf("PodDeploy error: %v", err)
		return
	}

	eveState := eve.Init(ctrl, devFirst)

	if err := checkAppState(ctrl, devUUID, podName, eveState, "RUNNING", 21*time.Minute); err != nil {
		t.Errorf("App state checking failed")
		return
	}

	ETestsFolder := filepath.Join(cfg.Eden.Root, "tests")
	sshCmd := fmt.Sprintf("ssh -o ConnectTimeout=10 -o StrictHostKeyChecking=no -o PasswordAuthentication=no -i %s/eclient/image/cert/id_rsa root@FWD_IP -p FWD_PORT ls", ETestsFolder)
	sshOut, err := withCapturingStdout(func() error {
		if err := ec.SdnForwardCmd("", "eth0", port, sshCmd+"/tst", cfg); err != nil {
			return fmt.Errorf("ssh to tst failed")
		}
		return nil
	})
	if err != nil {
		t.Error(err)
	}
	if err := checkOutput(string(sshOut), []string{"hello"}, []string{}); err != nil {
		t.Errorf("Ls to tst failed")
	}

	sshOut, err = withCapturingStdout(func() error {
		if err := ec.SdnForwardCmd("", "eth0", port, sshCmd+"/dir", cfg); err != nil {
			return fmt.Errorf("ssh to dir failed")
		}
		return nil
	})
	if err != nil {
		t.Error(err)
	}
	if err := checkOutput(string(sshOut), []string{"mount.txt"}, []string{}); err != nil {
		t.Errorf("ls to dir failed")
	}

	sshOut, err = withCapturingStdout(func() error {
		return ec.VolumeLs()
	})
	if err != nil {
		t.Error(err)
	}
	if err := checkOutput(string(sshOut), []string{"/dir", "/tst"}, []string{}); err != nil {
		t.Errorf("Volume ls failed")
	}

	volumeName := "eclient-mount_1_m_0"
	if err := ec.VolumeDetach(volumeName); err != nil {
		t.Errorf("Volume detach failed")
		return
	}

	if err = checkAppState(ctrl, devUUID, podName, eveState, "RUNNING", 15*time.Minute); err != nil {
		t.Errorf("eclient-mount RUNNING state InfoChecker: %v", err)
		return
	}

	sshOut, err = withCapturingStdout(func() error {
		if err := ec.SdnForwardCmd("", "eth0", port, sshCmd+"/dst", cfg); err != nil {
			return fmt.Errorf("ssh to dst failed")
		}
		return nil
	})
	if err != nil {
		t.Error(err)
	}
	if err := checkOutput(string(sshOut), []string{"hello"}, []string{}); err != nil {
		t.Errorf("Volume ls failed")
	}

	sshOut, err = withCapturingStdout(func() error {
		return ec.VolumeLs()
	})
	if err != nil {
		t.Error(err)
	}
	if err := checkOutput(string(sshOut), []string{"/dir", "/tst"}, []string{}); err != nil {
		t.Errorf("Volume ls failed")
	}

	if _, err := ec.PodDelete(podName, true); err != nil {
		t.Errorf("PodDelete failed")
		return
	}

	// check that podName was deleted
	if err := ec.ResetEve(cfg.Eve.CertsUUID); err != nil {
		t.Errorf("Resetting EVE failed")
		return
	}

	//sleep 30 secs

}
