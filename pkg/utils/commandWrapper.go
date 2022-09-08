package utils

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/lf-edge/eden/pkg/defaults"
	log "github.com/sirupsen/logrus"
)

//RunCommandBackground command run in goroutine
func RunCommandBackground(name string, logOutput io.Writer, args ...string) (pid int, err error) {
	cmd := exec.Command(name, args...)
	if logOutput != nil {
		stderr, err := cmd.StderrPipe()
		if err != nil {
			return 0, err
		}
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			return 0, err
		}
		go func() {
			merged := io.MultiReader(stderr, stdout)
			in := bufio.NewScanner(merged)
			for in.Scan() {
				_, err = logOutput.Write(in.Bytes())
				if err != nil {
					return
				}
			}
			if err := in.Err(); err != nil {
				log.Printf("error: %s", err)
			}
		}()
	}
	if err := cmd.Start(); err != nil {
		return 0, err
	}
	pid = cmd.Process.Pid
	go func() {
		log.Printf("Command finished with error: %v", cmd.Wait())
	}()
	return pid, nil
}

//RunCommandNohup run process in background
func RunCommandNohup(name string, logFile string, pidFile string, args ...string) (err error) {
	cmd := exec.Command(name, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if logFile != "" {
		var file io.Writer
		_, err := os.Stat(logFile)
		if !os.IsNotExist(err) {
			file, err = os.OpenFile(logFile, os.O_RDWR|os.O_APPEND, 0666)
		} else {
			file, err = os.Create(logFile)
		}
		if err != nil {
			return err
		}
		cmd.Stdout = file
		cmd.Stderr = file
	}
	if pidFile != "" {
		if _, err := os.Stat(pidFile); !os.IsNotExist(err) {
			if status, _ := StatusCommandWithPid(pidFile); strings.Contains(status, "running with pid") {
				// check if process with defined pid running
				return fmt.Errorf("pid file already exists")
			}
		}
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	if pidFile != "" {
		file, err := os.Create(pidFile)
		if err != nil {
			return err
		}
		if _, err = file.Write([]byte(strconv.Itoa(cmd.Process.Pid))); err != nil {
			return err
		}
		if err = file.Close(); err != nil {
			return err
		}
	}
	waiting := make(chan error)
	go func() {
		if err := cmd.Wait(); err != nil {
			if logFile != "" {
				if logFileContent, err := ioutil.ReadFile(logFile); err == nil {
					log.Errorf("log content: %s", strings.TrimSpace(string(logFileContent)))
				}
			}
			waiting <- err
		}
	}()
	select {
	case err := <-waiting:
		return fmt.Errorf("command %s failed with %s", name, err)
	case <-time.After(defaults.DefaultRepeatTimeout):
		return nil
	}
}

//StopCommandWithPid sends kill to pid from pidFile
func StopCommandWithPid(pidFile string) (err error) {
	content, err := ioutil.ReadFile(pidFile)
	if err != nil {
		return fmt.Errorf("cannot open pid file %s: %s", pidFile, err)
	}
	if err = os.Remove(pidFile); err != nil {
		return fmt.Errorf("cannot delete pid file %s: %s", pidFile, err)
	}
	pid, err := strconv.Atoi(string(content))
	if err != nil {
		return fmt.Errorf("cannot parse pid from file %s: %s", pidFile, err)
	}
	if err = syscall.Kill(pid, syscall.SIGKILL); err != nil {
		return fmt.Errorf("cannot kill process with pid: %d", pid)
	}

	return nil
}

//StatusCommandWithPid check if process with pid from pidFile running
func StatusCommandWithPid(pidFile string) (status string, err error) {
	content, err := ioutil.ReadFile(pidFile)
	if err != nil {
		return "process doesn't exist", nil
	}
	pid, err := strconv.Atoi(string(content))
	if err != nil {
		return "", fmt.Errorf("cannot parse pid from file %s: %s", pidFile, err)
	}
	if _, err = os.FindProcess(pid); err != nil {
		return "process not running", nil
	}
	if err = syscall.Kill(pid, syscall.Signal(0)); err != nil {
		return "process not running", nil
	}
	return fmt.Sprintf("running with pid %d", pid), nil
}

//RunCommandForeground command run in foreground
func RunCommandForeground(name string, args ...string) (err error) {
	return runCommandForeground(name, os.Stdin, args...)
}

//RunCommandForeground command run in foreground
func RunCommandForegroundWithStdin(name, stdin string, args ...string) (err error) {
	return runCommandForeground(name, strings.NewReader(stdin), args...)
}

func runCommandForeground(name string, stdin io.Reader, args ...string) (err error) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)
	cmd := exec.Command(name, args...)
	cmd.Stdin = stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	go func() {
		<-sigChan
		_ = cmd.Process.Kill()
	}()
	return cmd.Run()
}

//RunCommandAndWait run process in foreground
func RunCommandAndWait(name string, args ...string) (stdout string, stderr string, err error) {
	var stdoutBuffer bytes.Buffer
	var stderrBuffer bytes.Buffer
	cmd := exec.Command(name, args...)
	cmd.Stdout = &stdoutBuffer
	cmd.Stderr = &stderrBuffer
	err = cmd.Run()
	return stdoutBuffer.String(), stderrBuffer.String(), err
}

//RunCommandWithLogAndWait run process in foreground
func RunCommandWithLogAndWait(name string, logLevel log.Level, args ...string) (err error) {
	cmd := exec.Command(name, args...)
	if log.IsLevelEnabled(logLevel) {
		logWriter := log.StandardLogger().Out
		cmd.Stdout = logWriter
		cmd.Stderr = logWriter
	}
	return cmd.Run()
}