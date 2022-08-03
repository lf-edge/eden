package main

import (
	"flag"
	"net/http"

	"github.com/elazarl/goproxy"
	log "github.com/sirupsen/logrus"
)

// TODO: can we use TPROXY? https://github.com/FarFetchd/simple_tproxy_example

var AlwaysAllow goproxy.FuncHttpsHandler = func(host string, ctx *goproxy.ProxyCtx) (*goproxy.ConnectAction, string) {
	return goproxy.OkConnect, host
}

func main() {
	verbose := flag.Bool("v", false, "should every proxy request be logged to stdout")
	addr := flag.String("addr", ":8080", "proxy listen address")
	flag.Parse()
	if err := setCA(caCert, caKey); err != nil {
		log.Fatal(err)
	}
	proxy := goproxy.NewProxyHttpServer()
	proxy.OnRequest().HandleConnect(goproxy.AlwaysMitm)
	proxy.Verbose = *verbose
	log.Fatal(http.ListenAndServe(*addr, proxy))
}
