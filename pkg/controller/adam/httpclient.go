package adam

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/utils"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

// http client with correct config
func (adam *Ctx) getHTTPClient() *http.Client {
	tlsConfig := &tls.Config{}
	if adam.serverCA != "" {
		caCert, err := ioutil.ReadFile(adam.serverCA)
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

func (adam *Ctx) getObj(path string) (out string, err error) {
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

	response, err := repeatableAttempt(client, req)
	if err != nil {
		log.Fatalf("unable to send request: %v", err)
	}
	buf, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Printf("unable to read data from URL %s: %v", u, err)
		return "", err
	}
	return string(buf), nil
}

func (adam *Ctx) getList(path string) (out []string, err error) {
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

	response, err := repeatableAttempt(client, req)
	if err != nil {
		log.Fatalf("unable to send request: %v", err)
	}
	buf, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Printf("unable to read data from URL %s: %v", u, err)
		return nil, err
	}
	return strings.Fields(string(buf)), nil
}

func (adam *Ctx) postObj(path string, obj []byte) (err error) {
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
	req.Header.Set("Content-Type", "application/json")

	_, err = repeatableAttempt(client, req)
	if err != nil {
		log.Fatalf("unable to send request: %v", err)
	}
	return nil
}

func (adam *Ctx) putObj(path string, obj []byte) (err error) {
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
	_, err = repeatableAttempt(client, req)
	if err != nil {
		log.Fatalf("unable to send request: %v", err)
	}
	return nil
}

func repeatableAttempt(client *http.Client, req *http.Request) (response *http.Response, err error) {
	maxRepeat := defaults.DefaultRepeatCount
	delayTime := defaults.DefaultRepeatTimeout

	for i := 0; i < maxRepeat; i++ {
		timer := time.AfterFunc(2*delayTime, func() {
			i = 0
		})
		resp, err := client.Do(req)
		if err == nil {
			return resp, nil
		}
		log.Debugf("error %s URL %s: %v", req.Method, req.RequestURI, err)
		timer.Stop()
		log.Infof("Attempt to re-establish connection with controller (%d) of (%d)", i, maxRepeat)
		time.Sleep(delayTime)
	}
	return nil, fmt.Errorf("all connection attempts failed")
}
