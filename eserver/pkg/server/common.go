package server

import (
	"net/http"

	log "github.com/sirupsen/logrus"
)

const (
	contentType   = "Content-Type"
	mimeTextPlain = "text/plain"
)

func wrapError(err error, w http.ResponseWriter) {
	log.Error(err)
	w.Header().Add(contentType, mimeTextPlain)
	w.WriteHeader(http.StatusInternalServerError)
	_, _ = w.Write([]byte(err.Error()))
}
