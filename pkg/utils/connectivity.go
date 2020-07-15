package utils

import (
	"errors"
	"fmt"
	"github.com/dustin/go-humanize"
	log "github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

// writeCounter counts the number of bytes written to it. It implements to the io.Writer interface
// and we can pass this into io.TeeReader() which will report progress on each write cycle.
type writeCounter struct {
	message     string
	total       uint64
	beforePrint uint64
	step        uint64
}

//Write process bytes from downloader
func (wc *writeCounter) Write(p []byte) (int, error) {
	n := len(p)
	wc.total += uint64(n)
	wc.beforePrint += uint64(n)
	if wc.beforePrint > wc.step {
		wc.beforePrint = 0
		wc.printProgress()
	}
	return n, nil
}

func (wc writeCounter) printProgress() {
	if log.IsLevelEnabled(log.InfoLevel) {
		fmt.Printf("\r%s", strings.Repeat(" ", 35))
		fmt.Printf("\r%s %s complete", wc.message, humanize.Bytes(wc.total))
	}
}

//RequestHTTPWithTimeout make request to url with timeout
func RequestHTTPWithTimeout(url string, timeoutSeconds time.Duration) (string, error) {
	client := &http.Client{
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout: timeoutSeconds * time.Second,
			}).DialContext}}
	resp, err := client.Get(url)
	if err != nil {
		return "", fmt.Errorf("error in requestHTTP: %s", err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error in requestHTTP read response: %s", err)
	}
	return string(body), nil
}

//RequestHTTPRepeatWithTimeout make series of requests to url with timeout
//returnEmpty control if empty string is normal result
func RequestHTTPRepeatWithTimeout(url string, returnEmpty bool, timeoutSeconds time.Duration) (string, error) {
	done := make(chan string)
	quit := make(chan struct{})
	go func() {
		for {
			select {
			case <-quit:
				return
			default:
				if body, err := RequestHTTPWithTimeout(url, 5); err == nil {
					result := strings.TrimSpace(body)
					if returnEmpty || result != "" {
						done <- result
						return
					}
				}
			}
		}
	}()
	select {
	case result := <-done:
		return result, nil
	case <-time.After(timeoutSeconds * time.Second):
		close(quit)
		return "", errors.New("timeout")
	}
}

//DownloadFile download a url to a local file.
func DownloadFile(filepath string, url string) error {
	out, err := os.Create(filepath + ".tmp")
	if err != nil {
		return err
	}
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	counter := &writeCounter{step: 10 * 1024 * 1024, message: "Downloading..."}
	if _, err = io.Copy(out, io.TeeReader(resp.Body, counter)); err != nil {
		out.Close()
		return err
	}
	fmt.Printf("\n")
	if err = os.Rename(filepath+".tmp", filepath); err != nil {
		return err
	}
	return out.Close()
}
