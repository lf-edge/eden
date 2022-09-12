package main

import (
	"bufio"
	"bytes"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"

	"github.com/elazarl/goproxy"
	"github.com/inconshreveable/go-vhost"
	"github.com/lf-edge/eden/sdn/cmd/goproxy/config"
	log "github.com/sirupsen/logrus"
)

type tproxyResponseWriter struct {
	net.Conn
}

func (w tproxyResponseWriter) Header() http.Header {
	panic("Header() should not be called on this ResponseWriter")
}

func (w tproxyResponseWriter) Write(buf []byte) (int, error) {
	if bytes.Equal(buf, []byte("HTTP/1.0 200 OK\r\n\r\n")) {
		return len(buf), nil // throw away the HTTP OK response from the faux CONNECT request
	}
	return w.Conn.Write(buf)
}

func (w tproxyResponseWriter) WriteHeader(code int) {
	panic("WriteHeader() should not be called on this ResponseWriter")
}

func (w tproxyResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return w, bufio.NewReadWriter(bufio.NewReader(w), bufio.NewWriter(w)), nil
}

func tproxyListener(listener net.Listener, port uint16, httpsProxy *goproxy.ProxyHttpServer) {
	for {
		// Listen to the TLS ClientHello but make it a CONNECT request instead.
		c, err := listener.Accept()
		if err != nil {
			log.Errorf("Error accepting new connection: %v", err)
			continue
		}
		go func(c net.Conn) {
			log.Debugf("Received TLS ClientHello request on port %d", port)
			tlsConn, err := vhost.TLS(c)
			if err != nil {
				log.Errorf("Error accepting new connection: %v", err)
			}
			if tlsConn.Host() == "" {
				log.Errorf("Cannot support non-SNI enabled clients")
				return
			}
			log.Debugf("Received HTTPS request for host %s on port %d",
				tlsConn.Host(), port)

			connectReq := &http.Request{
				Method: "CONNECT",
				URL: &url.URL{
					Opaque: tlsConn.Host(),
					Host:   net.JoinHostPort(tlsConn.Host(), strconv.Itoa(int(port))),
				},
				Host:       tlsConn.Host(),
				Header:     make(http.Header),
				RemoteAddr: c.RemoteAddr().String(),
			}
			resp := tproxyResponseWriter{tlsConn}
			httpsProxy.ServeHTTP(resp, connectReq)
		}(c)
	}
}

func runTransparentProxy(proxyConfig config.ProxyConfig) {
	// Run HTTP proxy.
	if proxyConfig.HTTPPort != 0 {
		httpProxy := newProxy(proxyConfig)
		installProxyHandlers(proxyConfig, false, true, httpProxy)
		proxyAddr := fmt.Sprintf("%s:%d", proxyConfig.ListenIP, proxyConfig.HTTPPort)
		go func() {
			log.Fatalln(http.ListenAndServe(proxyAddr, httpProxy))
		}()
	}

	// Run HTTPS proxy(ies).
	if len(proxyConfig.HTTPSPorts) > 0 {
		for _, port := range proxyConfig.HTTPSPorts {
			httpsProxy := newProxy(proxyConfig)
			installProxyHandlers(proxyConfig, true, true, httpsProxy)
			go func(port uint16) {
				proxyAddr := fmt.Sprintf("%s:%d", proxyConfig.ListenIP, port)
				listener, err := net.Listen("tcp", proxyAddr)
				if err != nil {
					log.Fatalf("Error listening for HTTPS connections: %v", err)
				}
				tproxyListener(listener, port, httpsProxy)
			}(port)
		}
	}
}
