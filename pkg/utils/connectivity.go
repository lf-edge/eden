package utils

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"time"
)

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
				body, err := RequestHTTPWithTimeout(url, 5)
				if err == nil {
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
