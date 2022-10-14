package main

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"fmt"
	"math"
	"math/big"
	"net"
	"net/http"
	"time"

	"github.com/elazarl/goproxy"
	sdnapi "github.com/lf-edge/eden/sdn/vm/api"
	"github.com/lf-edge/eden/sdn/vm/cmd/goproxy/config"
	log "github.com/sirupsen/logrus"
)

func runExplicitProxy(proxyConfig config.ProxyConfig) {
	// Run HTTP proxy.
	if proxyConfig.HTTPPort.Port != 0 {
		httpProxy := newProxy(proxyConfig)
		installProxyHandlers(proxyConfig, false, false, httpProxy)
		runExplicitProxyListener(proxyConfig, proxyConfig.HTTPPort, httpProxy)
	}

	// Run HTTPS proxy(ies).
	for _, port := range proxyConfig.HTTPSPorts {
		httpsProxy := newProxy(proxyConfig)
		installProxyHandlers(proxyConfig, true, false, httpsProxy)
		runExplicitProxyListener(proxyConfig, port, httpsProxy)
	}
}

func runExplicitProxyListener(proxyConfig config.ProxyConfig, proxyPort sdnapi.ProxyPort,
	proxy *goproxy.ProxyHttpServer) {
	var listenHTTPS bool
	if proxyPort.ListenProto == sdnapi.ProxyListenProtoHTTPS {
		listenHTTPS = true
	}
	if listenHTTPS {
		ip := net.ParseIP(proxyConfig.ListenIP)
		if ip == nil {
			log.Fatalf("Failed to parse proxy IP: %v", proxyConfig.ListenIP)
		}
		go func(port uint16) {
			log.Fatalln(listenAndServeTLS(ip, port, proxyConfig.Hostname,
				[]byte(proxyConfig.CACertPEM), []byte(proxyConfig.CAKeyPEM),
				proxy))
		}(proxyPort.Port)
	} else {
		proxyAddr := fmt.Sprintf("%s:%d", proxyConfig.ListenIP, proxyPort.Port)
		go func() {
			log.Fatalln(http.ListenAndServe(proxyAddr, proxy))
		}()
	}
}

// From https://golang.org/src/net/http/server.go
// tcpKeepAliveListener sets TCP keep-alive timeouts on accepted
// connections. It's used by ListenAndServe and ListenAndServeTLS so
// dead TCP connections (e.g. closing laptop mid-download) eventually
// go away.
type tcpKeepAliveListener struct {
	*net.TCPListener
}

func (ln tcpKeepAliveListener) Accept() (c net.Conn, err error) {
	tc, err := ln.AcceptTCP()
	if err != nil {
		return
	}
	if err = tc.SetKeepAlive(true); err != nil {
		return
	}
	if err = tc.SetKeepAlivePeriod(3 * time.Minute); err != nil {
		return
	}
	return tc, nil
}

func listenAndServeTLS(ip net.IP, port uint16, hostname string,
	caCertPEM, caKeyPEM []byte, handler http.Handler) error {

	caCert, err := tls.X509KeyPair(caCertPEM, caKeyPEM)
	if err != nil {
		return errors.New("invalid public/private CA key pair")
	}
	serverCert, err := genServerCert(caCert, ip, hostname)
	if err != nil {
		return fmt.Errorf("failed to generate cert for %s/%s: %w",
			ip, hostname, err)
	}

	addr := fmt.Sprintf("%s:%d", ip.String(), port)
	server := &http.Server{Addr: addr, Handler: handler}

	config := &tls.Config{MinVersion: tls.VersionTLS12}
	config.NextProtos = []string{"http/1.1"}
	config.Certificates = make([]tls.Certificate, 1)
	config.Certificates[0] = *serverCert

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	tcpListener, ok := ln.(*net.TCPListener)
	if !ok {
		return fmt.Errorf("unexpected TCP listener type")
	}

	tlsListener := tls.NewListener(tcpKeepAliveListener{tcpListener},
		config)

	return server.Serve(tlsListener)
}

func genServerCert(ca tls.Certificate, ip net.IP, hostname string) (cert *tls.Certificate, err error) {
	var x509ca *x509.Certificate

	if x509ca, err = x509.ParseCertificate(ca.Certificate[0]); err != nil {
		return
	}
	start := time.Unix(0, 0)
	end, err := time.Parse("2006-01-02", "2049-12-31")
	if err != nil {
		return nil, err
	}

	serial, err := rand.Int(rand.Reader, big.NewInt(math.MaxInt64))
	if err != nil {
		return nil, err
	}
	template := x509.Certificate{
		SerialNumber: serial,
		Issuer:       x509ca.Subject,
		Subject: pkix.Name{
			Organization: []string{"LF Edge"},
		},
		NotBefore:             start,
		NotAfter:              end,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}
	if ip != nil {
		template.IPAddresses = append(template.IPAddresses, ip)
	}
	if hostname != "" {
		template.DNSNames = append(template.DNSNames, hostname)
		template.Subject.CommonName = hostname
	}

	var certpriv crypto.Signer
	switch ca.PrivateKey.(type) {
	case *rsa.PrivateKey:
		if certpriv, err = rsa.GenerateKey(rand.Reader, 2048); err != nil {
			return
		}
	case *ecdsa.PrivateKey:
		if certpriv, err = ecdsa.GenerateKey(elliptic.P256(), rand.Reader); err != nil {
			return
		}
	default:
		err = fmt.Errorf("unsupported key type %T", ca.PrivateKey)
		return
	}

	var derBytes []byte
	if derBytes, err = x509.CreateCertificate(rand.Reader, &template, x509ca,
		certpriv.Public(), ca.PrivateKey); err != nil {
		return
	}
	return &tls.Certificate{
		Certificate: [][]byte{derBytes, ca.Certificate[0]},
		PrivateKey:  certpriv,
	}, nil
}
