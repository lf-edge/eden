package server

import (
	"fmt"
	"github.com/gorilla/mux"
	"github.com/lf-edge/eden/eserver/pkg/manager"
	"log"
	"net/http"
)

type EServer struct {
	Port    string
	Address string
	Manager *manager.EServerManager
}

// log the request and client
func logRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("requested %s", r.URL.Path)
		next.ServeHTTP(w, r)
	})
}

//Start http server
//  /admin/list endpoint returns list of files
//  /admin/add-from-url endpoint fires download
//  /admin/status/{filename} returns fileinfo
//  /eserver/{filename} returns file
func (s *EServer) Start() {

	s.Manager.Init()

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
	ad.HandleFunc("/add-from-url", admin.addFromUrl).Methods("POST")
	ad.HandleFunc("/add-from-file", admin.addFromFile).Methods("POST")
	ad.HandleFunc("/status/{filename}", admin.getFileStatus).Methods("GET")

	router.HandleFunc("/eserver/{filename}", api.getFile).Methods("GET")

	server := &http.Server{
		Handler: router,
		Addr:    fmt.Sprintf("%s:%s", s.Address, s.Port),
	}

	log.Println("Starting eserver:")
	log.Printf("\tIP:Port: %s:%s\n", s.Address, s.Port)
	log.Printf("\tDirectory: %s\n", s.Manager.Dir)
	log.Fatal(server.ListenAndServe())
}
