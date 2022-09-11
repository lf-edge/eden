package main

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/elazarl/goproxy"
	"github.com/elazarl/goproxy/ext/auth"
	sdnapi "github.com/lf-edge/eden/sdn/api"
	"github.com/lf-edge/eden/sdn/cmd/goproxy/config"
	log "github.com/sirupsen/logrus"
)

const (
	authRealm = "Auth"
)

var proxy *goproxy.ProxyHttpServer

func dstPortIs(port uint16) goproxy.ReqConditionFunc {
	return func(req *http.Request, ctx *goproxy.ProxyCtx) bool {
		dstHost := strings.Split(req.URL.Host, ":")
		if len(dstHost) > 1 {
			dstPort, err := strconv.Atoi(dstHost[1])
			if err != nil {
				log.Errorf("Failed to convert dst port: %v", err)
				return false
			}
			log.Debugf("dstPortIs %v: %d vs. %d", req, dstPort, port)
			return dstPort == int(port)
		}
		return false
	}
}

func dstHostIs(host string) goproxy.ReqConditionFunc {
	return func(req *http.Request, ctx *goproxy.ProxyCtx) bool {
		if host == "" {
			// default rule matching any host
			return true
		}
		dstHost := strings.Split(req.URL.Host, ":")[0]
		log.Debugf("dstHostIs %v: %s vs. %s", req, dstHost, host)
		return dstHost == host
	}
}

func isConnect(req *http.Request, ctx *goproxy.ProxyCtx) bool {
	_, hasConnectMark := ctx.UserData.(connectMark)
	log.Debugf("isConnect %v: %t", req, hasConnectMark)
	return hasConnectMark
}

func forwardConnect(host string, ctx *goproxy.ProxyCtx) (*goproxy.ConnectAction, string) {
	log.Debugf("forwardConnect: %s", host)
	return goproxy.OkConnect, host
}

func mitmHTTPConnect(host string, ctx *goproxy.ProxyCtx) (*goproxy.ConnectAction, string) {
	log.Debugf("mitmHTTPConnect: %s", host)
	return goproxy.HTTPMitmConnect, host
}

type connectMark struct {
	session int64
}

func markConnect(host string, ctx *goproxy.ProxyCtx) (*goproxy.ConnectAction, string) {
	ctx.UserData = connectMark{
		session: ctx.Session,
	}
	return nil, host // continue with other handlers
}

func basicAuthForConnect(f func(user, passwd string) bool) goproxy.HttpsHandler {
	authHandler := auth.BasicConnect(authRealm, f)
	return goproxy.FuncHttpsHandler(
		func(host string, ctx *goproxy.ProxyCtx) (*goproxy.ConnectAction, string) {
			log.Debugf("basicAuthForConnect: %s", host)
			action, host := authHandler.HandleConnect(host, ctx)
			if action == goproxy.OkConnect {
				// Return nil action, do not overshadow other handlers in the queue.
				return nil, host
			}
			return action, host
		})
}

func forwardHTTP(req *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
	log.Debugf("forwardHTTP: %v", req)
	resp, err := ctx.RoundTrip(req)
	if err != nil {
		return req, goproxy.NewResponse(req,
			goproxy.ContentTypeText, http.StatusInternalServerError, err.Error())
	}
	return req, resp
}

func rejectHTTP(req *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
	log.Debugf("rejectHTTP: %v", req)
	return req, goproxy.NewResponse(req,
		goproxy.ContentTypeText, http.StatusForbidden, "Forbidden by proxy!")
}

func setCA(caCert, caKey string) error {
	goproxyCa, err := tls.X509KeyPair([]byte(caCert), []byte(caKey))
	if err != nil {
		return err
	}
	if goproxyCa.Leaf, err = x509.ParseCertificate(goproxyCa.Certificate[0]); err != nil {
		return err
	}
	goproxy.GoproxyCa = goproxyCa
	goproxy.OkConnect = &goproxy.ConnectAction{
		Action: goproxy.ConnectAccept, TLSConfig: goproxy.TLSConfigFromCA(&goproxyCa)}
	goproxy.MitmConnect = &goproxy.ConnectAction{
		Action: goproxy.ConnectMitm, TLSConfig: goproxy.TLSConfigFromCA(&goproxyCa)}
	goproxy.HTTPMitmConnect = &goproxy.ConnectAction{
		Action: goproxy.ConnectHTTPMitm, TLSConfig: goproxy.TLSConfigFromCA(&goproxyCa)}
	goproxy.RejectConnect = &goproxy.ConnectAction{
		Action: goproxy.ConnectReject, TLSConfig: goproxy.TLSConfigFromCA(&goproxyCa)}
	return nil
}

func installHandler(rule sdnapi.ProxyRule) {
	notConnect := goproxy.Not(goproxy.ReqConditionFunc(isConnect))
	isHTTP := dstPortIs(80)
	isHTTPS := goproxy.Not(isHTTP)
	switch rule.Action {
	case sdnapi.PxForward:
		// Forward CONNECT (works for both HTTP and HTTPS)
		proxy.OnRequest(dstHostIs(rule.ReqHost)).HandleConnect(
			goproxy.FuncHttpsHandler(forwardConnect))
		// Forward HTTP GET, POST, etc. (but not CONNECT)
		proxy.OnRequest(dstHostIs(rule.ReqHost), notConnect).DoFunc(
			forwardHTTP)

	case sdnapi.PxReject:
		// Reject CONNECT
		proxy.OnRequest(dstHostIs(rule.ReqHost)).HandleConnect(
			goproxy.AlwaysReject)
		// Reject HTTP GET, POST, etc. (but not CONNECT)
		proxy.OnRequest(dstHostIs(rule.ReqHost), notConnect).DoFunc(
			rejectHTTP)

	case sdnapi.PxMITM:
		// CONNECT is plain HTTP tunneling
		proxy.OnRequest(dstHostIs(rule.ReqHost), isHTTP).HandleConnect(
			goproxy.FuncHttpsHandler(mitmHTTPConnect))
		// CONNECT is TLS
		proxy.OnRequest(dstHostIs(rule.ReqHost), isHTTPS).HandleConnect(
			goproxy.AlwaysMitm)
		// non-CONNECT methods are simply forwarded
		proxy.OnRequest(dstHostIs(rule.ReqHost), notConnect).DoFunc(
			forwardHTTP)

	}
}

func main() {
	log.SetReportCaller(true)
	configFile := flag.String("c", "/etc/goproxy.conf", "proxy config file")
	flag.Parse()

	// Instantiate proxy.
	proxy = goproxy.NewProxyHttpServer()
	proxy.NonproxyHandler = http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Host == "" {
			fmt.Fprintln(w, "Cannot handle requests without Host header, e.g., HTTP 1.0")
			return
		}
		req.URL.Scheme = "http"
		req.URL.Host = req.Host
		proxy.ServeHTTP(w, req)
	})
	proxy.Logger = log.StandardLogger()

	// Read and parse config file.
	configBytes, err := ioutil.ReadFile(*configFile)
	if err != nil {
		log.Fatalf("failed to read config file %s: %v", *configFile, err)
	}
	var proxyConfig config.ProxyConfig
	if err = json.Unmarshal(configBytes, &proxyConfig); err != nil {
		log.Fatalf("failed to unmarshal proxy config: %v", err)
	}

	// Process proxy config.
	if proxyConfig.LogFile != "" {
		logFile, err := os.OpenFile(proxyConfig.LogFile, os.O_WRONLY|os.O_CREATE, 0755)
		if err != nil {
			log.Fatalf("failed to open log file %s: %v", proxyConfig.LogFile, err)
		}
		log.SetOutput(logFile)
	}
	proxy.Verbose = proxyConfig.Verbose
	if proxyConfig.Verbose {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}
	if proxyConfig.PidFile != "" {
		pidBytes := []byte(fmt.Sprintf("%d", os.Getpid()))
		err = ioutil.WriteFile(proxyConfig.PidFile, pidBytes, 0664)
		if err != nil {
			log.Fatalf("failed to write PID file %s: %v", proxyConfig.PidFile, err)
		}
		defer os.Remove(proxyConfig.PidFile)
	}
	if proxyConfig.CACertPEM != "" {
		log.Infof("Installing CA cert and key")
		if err = setCA(proxyConfig.CACertPEM, proxyConfig.CAKeyPEM); err != nil {
			log.Fatal(err)
		}
	}
	if len(proxyConfig.Users) > 0 {
		userAuth := func(user, passwd string) bool {
			for i := range proxyConfig.Users {
				if proxyConfig.Users[i].Username == user {
					return proxyConfig.Users[i].Password == passwd
				}
			}
			// No such user.
			return false
		}
		log.Infof("Installing handlers for basic auth")
		proxy.OnRequest().Do(auth.Basic(authRealm, userAuth))
		proxy.OnRequest().HandleConnect(basicAuthForConnect(userAuth))
	}

	// Prepare proxy handlers.
	proxy.OnRequest().HandleConnect(goproxy.FuncHttpsHandler(markConnect))
	var defaultRule *sdnapi.ProxyRule
	for _, rule := range proxyConfig.ProxyRules {
		if rule.ReqHost == "" {
			defaultRule = &rule
			continue
		}
		installHandler(rule)
	}
	if defaultRule != nil {
		installHandler(*defaultRule)
	}

	// Run HTTP(S) proxy.
	proxyAddr := fmt.Sprintf("%s:%d", proxyConfig.ListenIP, proxyConfig.ListenPort)
	log.Infof("Proxy listening on the address %s", proxyAddr)
	log.Fatal(http.ListenAndServe(proxyAddr, proxy))
}
