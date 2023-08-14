package adam

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/lf-edge/eden/pkg/utils"
	log "github.com/sirupsen/logrus"
)

// http client with correct config
func (adam *Ctx) getHTTPClient() *http.Client {
	tlsConfig := &tls.Config{}
	if adam.serverCA != "" {
		caCert, err := os.ReadFile(adam.serverCA)
		if err != nil {
			log.Fatalf("unable to read server CA file at %s: %v", adam.serverCA, err)
		}
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)
		tlsConfig.RootCAs = caCertPool
	}
	if adam.insecureTLS {
		tlsConfig.InsecureSkipVerify = true
	}
	var client = &http.Client{
		Timeout: time.Second * 10,
		Transport: &http.Transport{
			TLSClientConfig:       tlsConfig,
			TLSHandshakeTimeout:   10 * time.Second,
			ResponseHeaderTimeout: 10 * time.Second,
		},
	}
	return client
}

func (adam *Ctx) deleteObj(path string) (err error) {
	u, err := utils.ResolveURL(adam.url, path)
	if err != nil {
		log.Printf("error constructing URL: %v", err)
		return err
	}
	client := adam.getHTTPClient()
	req, err := http.NewRequest("DELETE", u, nil)
	if err != nil {
		log.Fatalf("unable to create new http request: %v", err)
	}

	response, err := utils.RepeatableAttempt(client, req)
	if err != nil {
		log.Fatalf("unable to send request: %v", err)
	}
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("status code: %d", response.StatusCode)
	}
	return nil
}

func (adam *Ctx) getObj(path string, acceptMime string) (out string, err error) {
	u, err := utils.ResolveURL(adam.url, path)
	if err != nil {
		log.Printf("error constructing URL: %v", err)
		return "", err
	}
	client := adam.getHTTPClient()
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		log.Fatalf("unable to create new http request: %v", err)
	}
	if acceptMime != "" {
		req.Header.Set("Accept", acceptMime)
	}

	response, err := utils.RepeatableAttempt(client, req)
	if err != nil {
		log.Fatalf("unable to send request: %v", err)
	}
	buf, err := io.ReadAll(response.Body)
	if err != nil {
		log.Printf("unable to read data from URL %s: %v", u, err)
		return "", err
	}
	return string(buf), nil
}

func (adam *Ctx) getList(path string, acceptMime string) (out []string, err error) {
	u, err := utils.ResolveURL(adam.url, path)
	if err != nil {
		log.Printf("error constructing URL: %v", err)
		return nil, err
	}
	client := adam.getHTTPClient()
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		log.Fatalf("unable to create new http request: %v", err)
	}
	if acceptMime != "" {
		req.Header.Set("Accept", acceptMime)
	}

	response, err := utils.RepeatableAttempt(client, req)
	if err != nil {
		log.Fatalf("unable to send request: %v", err)
	}
	buf, err := io.ReadAll(response.Body)
	if err != nil {
		log.Printf("unable to read data from URL %s: %v", u, err)
		return nil, err
	}
	return strings.Fields(string(buf)), nil
}

func (adam *Ctx) postObj(path string, obj []byte, mimeType string) (err error) {
	u, err := utils.ResolveURL(adam.url, path)
	if err != nil {
		log.Printf("error constructing URL: %v", err)
		return err
	}
	client := adam.getHTTPClient()
	req, err := http.NewRequest("POST", u, bytes.NewBuffer(obj))
	if err != nil {
		log.Fatalf("unable to create new http request: %v", err)
	}
	req.Header.Set("Content-Type", mimeType)

	_, err = utils.RepeatableAttempt(client, req)
	if err != nil {
		log.Fatalf("unable to send request: %v", err)
	}
	return nil
}

func (adam *Ctx) putObj(path string, obj []byte, mimeType string) (err error) {
	u, err := utils.ResolveURL(adam.url, path)
	if err != nil {
		log.Printf("error constructing URL: %v", err)
		return err
	}
	client := adam.getHTTPClient()
	req, err := http.NewRequest("PUT", u, bytes.NewBuffer(obj))
	if err != nil {
		log.Fatalf("unable to create new http request: %v", err)
	}
	req.Header.Set("Content-Type", mimeType)
	_, err = utils.RepeatableAttempt(client, req)
	if err != nil {
		log.Fatalf("unable to send request: %v", err)
	}
	return nil
}
