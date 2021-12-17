package server

import (
	"fmt"
	"net"
	"net/http"

	"github.com/gorilla/mux"
)

func (s *EServer) serveHTTP(listener net.Listener, errorChan chan error) {
	api := &apiHandler{
		manager: s.Manager,
	}

	admin := &adminHandler{
		manager: s.Manager,
	}

	router := mux.NewRouter()

	ad := router.PathPrefix("/admin").Subrouter()

	router.Use(logRequest)

	ad.HandleFunc("/list", admin.list).Methods("GET")
	ad.HandleFunc("/add-from-url", admin.addFromURL).Methods("POST")
	ad.HandleFunc("/add-from-file", admin.addFromFile).Methods("POST")
	ad.HandleFunc("/status/{filename:[A-Za-z0-9_\\-.\\/]*}", admin.getFileStatus).Methods("GET")

	router.HandleFunc("/eserver/{filename:[A-Za-z0-9_\\-.\\/]*}", api.getFile).Methods("GET")

	server := &http.Server{
		Handler: router,
		Addr:    fmt.Sprintf("%s:%s", s.Address, s.Port),
	}
	errorChan <- server.Serve(listener)
}
