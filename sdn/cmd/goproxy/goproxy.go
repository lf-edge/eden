package main

import (
	"log"
	"net/http"

	"github.com/elazarl/goproxy"
)

// TODO: can we use TPROXY? https://github.com/FarFetchd/simple_tproxy_example

// TODO: consider/try also this: https://github.com/snail007/goproxy (accidentally the same name but completely different project)

func main() {
	proxy := goproxy.NewProxyHttpServer()
	proxy.Verbose = true
	log.Fatal(http.ListenAndServe(":8080", proxy))
}
