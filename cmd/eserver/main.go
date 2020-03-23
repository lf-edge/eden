package main

import (
	"flag"
	"log"
	"net/http"
)

func main() {
	port := flag.String("p", "8888", "port to serve on")
	directory := flag.String("d", ".", "the directory with static files")
	flag.Parse()

	http.Handle("/", http.FileServer(http.Dir(*directory)))

	log.Printf("Serving %s on HTTP port: %s\n", *directory, *port)
	log.Fatal(http.ListenAndServe(":"+*port, nil))
}
