package main

import (
	"flag"
	"fmt"
	"net"
	"net/http"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

const (
	defaultPort = 6666
)

func logRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Infof("HTTP request %s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}

func main() {
	debug := flag.Bool("debug", false, "Set Debug log level")
	port := flag.Uint("port", defaultPort, "Port on which to listen")
	ip := flag.String("ip", "", "IP address on which to listen")
	flag.Parse()

	if *debug {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}
	log.SetReportCaller(true)

	agent := &agent{}
	if err := agent.init(); err != nil {
		log.Fatal(err)
	}

	router := mux.NewRouter()
	router.Use(logRequest)

	router.HandleFunc("/net-model.json", agent.getNetModel).Methods("GET")
	router.HandleFunc("/net-model.json", agent.applyNetModel).Methods("PUT")
	router.HandleFunc("/net-config.gv", agent.getNetConfig).Methods("GET")
	router.HandleFunc("/sdn-status.json", agent.getSDNStatus).Methods("GET")
	// TODO: metrics?

	srv := &http.Server{
		Handler: router,
		Addr:    net.JoinHostPort(*ip, fmt.Sprintf("%d", *port)),
	}
	log.Fatal(srv.ListenAndServe())
}
