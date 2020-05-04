package adam

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"github.com/lf-edge/eden/pkg/utils"
	"io/ioutil"
	"log"
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
	response, err := client.Get(u)
	if err != nil {
		log.Printf("error reading URL %s: %v", u, err)
		return "", err
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
	response, err := client.Get(u)
	if err != nil {
		log.Printf("error reading URL %s: %v", u, err)
		return nil, err
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
	_, err = client.Post(u, "application/json", bytes.NewBuffer(obj))
	if err != nil {
		log.Printf("unable to post data to URL %s: %v", u, err)
		return err
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
		log.Printf("unable to create new http request: %v", err)
		return err
	}
	_, err = client.Do(req)
	if err != nil {
		log.Printf("error PUT URL %s: %v", u, err)
		return err
	}
	return nil
}
