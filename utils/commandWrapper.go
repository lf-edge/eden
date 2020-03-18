package utils

import (
	"bufio"
	"bytes"
	"io"
	"log"
	"os/exec"
)

//RunCommandBackground command run in goroutine
func RunCommandBackground(name string, logOutput bool, args ...string) (pid int, err error) {
	cmd := exec.Command(name, args...)
	if logOutput {
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
				log.Printf(in.Text())
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
