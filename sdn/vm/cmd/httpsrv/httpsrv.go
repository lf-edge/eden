package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	sdnapi "github.com/lf-edge/eden/sdn/vm/api"
	"github.com/lf-edge/eden/sdn/vm/cmd/httpsrv/config"
	log "github.com/sirupsen/logrus"
)

func handler(content sdnapi.HTTPContent) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Debugf("Received request: %+v", r)
		w.Header().Add("Content-Type", content.ContentType)
		_, err := w.Write([]byte(content.Content))
		if err != nil {
			log.Errorf("Failed to write content for request %+v: %v", r, err)
		}
	}
}

func main() {
	log.SetReportCaller(true)
	configFile := flag.String("c", "/etc/httpsrv.conf", "HTTP server config file")
	flag.Parse()

	// Read and parse config file.
	configBytes, err := ioutil.ReadFile(*configFile)
	if err != nil {
		log.Fatalf("failed to read config file %s: %v", *configFile, err)
	}
	var httpSrvConfig config.HttpSrvConfig
	if err = json.Unmarshal(configBytes, &httpSrvConfig); err != nil {
		log.Fatalf("failed to unmarshal HTTP server config: %v", err)
	}

	// Process HTTP server config.
	if httpSrvConfig.LogFile != "" {
		logFile, err := os.OpenFile(httpSrvConfig.LogFile, os.O_WRONLY|os.O_CREATE, 0755)
		if err != nil {
			log.Fatalf("failed to open log file %s: %v", httpSrvConfig.LogFile, err)
		}
		log.SetOutput(logFile)
	}
	if httpSrvConfig.Verbose {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}
	if httpSrvConfig.PidFile != "" {
		pidBytes := []byte(fmt.Sprintf("%d", os.Getpid()))
		err = ioutil.WriteFile(httpSrvConfig.PidFile, pidBytes, 0664)
		if err != nil {
			log.Fatalf("failed to write PID file %s: %v", httpSrvConfig.PidFile, err)
		}
		defer os.Remove(httpSrvConfig.PidFile)
	}

	for path, content := range httpSrvConfig.Paths {
		http.HandleFunc(path, handler(content))
	}

	if httpSrvConfig.HTTPPort != 0 {
		srvAddr := fmt.Sprintf("%s:%d", httpSrvConfig.ListenIP, httpSrvConfig.HTTPPort)
		go func() {
			log.Debugf("HTTP server listening on %s", srvAddr)
			log.Fatalln(http.ListenAndServe(srvAddr, nil))
		}()
	}

	if httpSrvConfig.HTTPSPort != 0 {
		srvAddr := fmt.Sprintf("%s:%d", httpSrvConfig.ListenIP, httpSrvConfig.HTTPSPort)
		certFile, err := os.CreateTemp("", "httpsrv-*.cert")
		if err != nil {
			log.Fatalf("failed to create temporary file for the certificate: %v", err)
		}
		keyFile, err := os.CreateTemp("", "httpsrv-*.key")
		if err != nil {
			log.Fatalf("failed to create temporary file for the key: %v", err)
		}
		defer func() {
			if err = os.Remove(certFile.Name()); err != nil {
				log.Warnf("failed to remove temporary file %s: %v", certFile.Name(), err)
			}
			if err = os.Remove(keyFile.Name()); err != nil {
				log.Warnf("failed to remove temporary file %s: %v", keyFile.Name(), err)
			}
		}()
		if _, err = certFile.WriteString(httpSrvConfig.CertPEM); err != nil {
			log.Fatalf("failed to write server cert to file %s: %v", certFile.Name(), err)
		}
		if _, err = keyFile.WriteString(httpSrvConfig.KeyPEM); err != nil {
			log.Fatalf("failed to write server key to file %s: %v", keyFile.Name(), err)
		}
		log.Debugf("Storing server certificate to file %s", certFile.Name())
		log.Debugf("Storing server key to file %s", keyFile.Name())
		go func() {
			log.Debugf("HTTPS server listening on %s", srvAddr)
			log.Fatalln(http.ListenAndServeTLS(srvAddr, certFile.Name(), keyFile.Name(), nil))
		}()
	}

	cancelChan := make(chan os.Signal, 1)
	// Catch termination or interrupt signal.
	signal.Notify(cancelChan, syscall.SIGTERM, syscall.SIGINT)
	sig := <-cancelChan
	log.Infof("Caught terimation/interrupt signal: %v, exiting...", sig)
}
