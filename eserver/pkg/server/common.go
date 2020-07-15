package server

import (
	log "github.com/sirupsen/logrus"
	"net/http"
)

const (
	contentType   = "Content-Type"
	mimeTextPlain = "text/plain"
)

func wrapError(err error, w http.ResponseWriter) {
	log.Error(err)
	w.Header().Add(contentType, mimeTextPlain)
	w.WriteHeader(http.StatusInternalServerError)
	w.Write([]byte(err.Error()))
}
