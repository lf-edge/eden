package openevec_test

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
	"testing"

	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/openevec"
	"github.com/onsi/gomega"
)

// startProcess starts a long-running process standing in for QEMU or swtpm
// and writes its pid into pidFile, as eden does when it starts them.
func startProcess(t *testing.T, pidFile string) *exec.Cmd {
	t.Helper()
	cmd := exec.Command("sleep", "60")
	if err := cmd.Start(); err != nil {
		t.Fatalf("cannot start process: %v", err)
	}
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	})
	if err := os.MkdirAll(filepath.Dir(pidFile), 0o755); err != nil {
		t.Fatalf("cannot create directory for %s: %v", pidFile, err)
	}
	if err := os.WriteFile(pidFile, []byte(strconv.Itoa(cmd.Process.Pid)), 0o644); err != nil {
		t.Fatalf("cannot write %s: %v", pidFile, err)
	}
	return cmd
}

// expectKilled waits for cmd and checks that it was ended by SIGKILL.
func expectKilled(g *gomega.WithT, cmd *exec.Cmd) {
	err := cmd.Wait()
	var exitErr *exec.ExitError
	g.Expect(errors.As(err, &exitErr)).To(gomega.BeTrue(), "process was not stopped: %v", err)
	status, ok := exitErr.Sys().(syscall.WaitStatus)
	g.Expect(ok).To(gomega.BeTrue())
	g.Expect(status.Signaled()).To(gomega.BeTrue())
	g.Expect(status.Signal()).To(gomega.Equal(syscall.SIGKILL))
}

func TestStopEveQemuWithTPM(t *testing.T) {
	t.Parallel()

	g := gomega.NewGomegaWithT(t)

	imagesDir := t.TempDir()
	cfg := &openevec.EdenSetupArgs{}
	cfg.Eve.DevModel = defaults.DefaultQemuModel
	cfg.Eve.ImageFile = filepath.Join(imagesDir, "eve", "live.img")
	cfg.Eve.Pid = filepath.Join(t.TempDir(), "eve.pid")
	cfg.Eve.TPM = true
	cfg.Sdn.Disable = true

	eve := startProcess(t, cfg.Eve.Pid)
	// swtpm keeps its state, and so its pid file, in a "swtpm" directory
	// next to the EVE image.
	swtpmPidFile := filepath.Join(imagesDir, "eve", "swtpm", "swtpm.pid")
	swtpm := startProcess(t, swtpmPidFile)

	g.Expect(openevec.CreateOpenEVEC(cfg).StopEve(defaults.DefaultVBoxVMName)).To(gomega.Succeed())

	expectKilled(g, eve)
	expectKilled(g, swtpm)
	g.Expect(cfg.Eve.Pid).NotTo(gomega.BeAnExistingFile())
	g.Expect(swtpmPidFile).NotTo(gomega.BeAnExistingFile())
}
