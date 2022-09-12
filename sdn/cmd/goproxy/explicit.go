package main

import (
	"fmt"
	"net/http"

	"github.com/lf-edge/eden/sdn/cmd/goproxy/config"
	log "github.com/sirupsen/logrus"
)

func runExplicitProxy(proxyConfig config.ProxyConfig) {
	// Run HTTP proxy.
	if proxyConfig.HTTPPort != 0 {
		httpProxy := newProxy(proxyConfig)
		installProxyHandlers(proxyConfig, false, false, httpProxy)
		proxyAddr := fmt.Sprintf("%s:%d", proxyConfig.ListenIP, proxyConfig.HTTPPort)
		go func() {
			log.Fatalln(http.ListenAndServe(proxyAddr, httpProxy))
		}()
	}

	// Run HTTPS proxy(ies).
	if len(proxyConfig.HTTPSPorts) > 0 {
		for _, port := range proxyConfig.HTTPSPorts {
			httpsProxy := newProxy(proxyConfig)
			installProxyHandlers(proxyConfig, true, false, httpsProxy)
			go func(port uint16) {
				proxyAddr := fmt.Sprintf("%s:%d", proxyConfig.ListenIP, port)
				log.Fatalln(http.ListenAndServe(proxyAddr, httpsProxy))
			}(port)
		}
	}
}
