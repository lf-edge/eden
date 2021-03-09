package server

import (
	"fmt"
	"log"
	"net"
	"net/http"

	"github.com/lf-edge/eden/eserver/pkg/manager"
)

//EServer stores info about settings
type EServer struct {
	Port     string
	Address  string
	Manager  *manager.EServerManager
	User     string
	Password string
	ReadOnly bool
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

	log.Println("Starting eserver:")
	log.Printf("\tIP:Port: %s:%s\n", s.Address, s.Port)
	log.Printf("\tDirectory: %s\n", s.Manager.Dir)

	// server both services (sftp and http) on the same port
	l, err := net.Listen("tcp", fmt.Sprintf("%s:%s", s.Address, s.Port))
	if err != nil {
		log.Fatalf("net.Listen error: %s", err)
	}
	sshListener, httpListener := MuxListener(l)
	errorChan := make(chan error)
	go s.serveHTTP(httpListener, errorChan)
	go s.serveSFTP(sshListener, errorChan)
	log.Println(<-errorChan)
}
