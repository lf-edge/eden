package utils

import (
	"bufio"
	"bytes"
	"io"
	"log"
	"os"
	"os/exec"
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
	err = cmd.Start()
	if err != nil {
		return 0, err
	}
	pid = cmd.Process.Pid
	go func() {
		err = cmd.Wait()
		log.Printf("Command finished with error: %v", err)
	}()
	return pid, nil
}

//RunCommandForeground command run in foreground
func RunCommandForeground(name string, args ...string) (err error) {
	cmd := exec.Command(name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Pdeathsig: syscall.SIGTERM,
	}
	err = cmd.Run()
	if err != nil {
		return
	}
	defer cmd.Process.Kill()
	return cmd.Wait()
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

//RunCommandWithSTDINAndWait run process in foreground with stdin passed as arg
func RunCommandWithSTDINAndWait(name string, stdin string, args ...string) (stdout string, stderr string, err error) {
	var stdoutBuffer bytes.Buffer
	var stderrBuffer bytes.Buffer
	cmd := exec.Command(name, args...)
	cmd.Stdin = strings.NewReader(stdin)
	cmd.Stdout = &stdoutBuffer
	cmd.Stderr = &stderrBuffer
	err = cmd.Run()
	return stdoutBuffer.String(), stderrBuffer.String(), err
}
