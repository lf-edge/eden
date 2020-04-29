package utils

import (
	"bufio"
	"bytes"
	"fmt"
	log "github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
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
			return fmt.Errorf("pid file already exists")
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
	return nil
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
		return "no pid file", nil
	}
	pid, err := strconv.Atoi(string(content))
	if err != nil {
		return "", fmt.Errorf("cannot parse pid from file %s: %s", pidFile, err)
	}
	if _, err = os.FindProcess(pid); err != nil {
		return "not found", nil
	}
	if err = syscall.Kill(pid, syscall.Signal(0)); err != nil {
		return "not found", nil
	}
	return "running", nil
}

//RunCommandForeground command run in foreground
func RunCommandForeground(name string, args ...string) (err error) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)
	cmd := exec.Command(name, args...)
	cmd.Stdin = os.Stdin
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
	return stdoutBuffer.String(), stderrBuffer.String(), cmd.Run()
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

//RunCommandWithSTDINAndWait run process in foreground with stdin passed as arg
func RunCommandWithSTDINAndWait(name string, stdin string, args ...string) (stdout string, stderr string, err error) {
	var stdoutBuffer bytes.Buffer
	var stderrBuffer bytes.Buffer
	cmd := exec.Command(name, args...)
	cmd.Stdin = strings.NewReader(stdin)
	cmd.Stdout = &stdoutBuffer
	cmd.Stderr = &stderrBuffer
	return stdoutBuffer.String(), stderrBuffer.String(), cmd.Run()
}
