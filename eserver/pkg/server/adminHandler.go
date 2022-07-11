package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/lf-edge/eden/eserver/api"
	"github.com/lf-edge/eden/eserver/pkg/manager"
	log "github.com/sirupsen/logrus"
)

type adminHandler struct {
	manager *manager.EServerManager
}

func (h *adminHandler) list(w http.ResponseWriter, _ *http.Request) {
	files := h.manager.ListFileNames()
	w.Header().Add(contentType, mimeTextPlain)
	w.WriteHeader(http.StatusOK)
	for _, value := range files {
		fileName := bytes.NewBufferString(value + "\n")
		if _, err := fileName.WriteTo(w); err != nil {
			wrapError(err, w)
			return
		}
	}
}

func (h *adminHandler) addFromURL(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var data api.URLArg
	err := decoder.Decode(&data)
	if err != nil {
		wrapError(err, w)
		return
	}
	name, err := h.manager.AddFile(data.URL)
	if err != nil {
		wrapError(err, w)
		return
	}
	w.Header().Add(contentType, mimeTextPlain)
	w.WriteHeader(http.StatusCreated)
	_, _ = w.Write([]byte(name))
}

func (h *adminHandler) addFromFile(w http.ResponseWriter, r *http.Request) {
	reader, err := r.MultipartReader()
	if err != nil {
		wrapError(err, w)
		return
	}
	part, err := reader.NextPart()
	if err != nil {
		wrapError(err, w)
		return
	}
	if part.FormName() != "file" {
		wrapError(fmt.Errorf("wrong form"), w)
		return
	}
	defer part.Close()
	fileInfo := h.manager.AddFileFromMultipart(part)
	if fileInfo.Error != "" {
		log.Error(fileInfo.Error)
	}
	out, err := json.Marshal(fileInfo)
	if err != nil {
		wrapError(err, w)
		return
	}
	w.Header().Add(contentType, mimeTextPlain)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(out)
}

func (h *adminHandler) getFileStatus(w http.ResponseWriter, r *http.Request) {
	u := mux.Vars(r)["filename"]
	fileInfo := h.manager.GetFileInfo(u)
	out, err := json.Marshal(fileInfo)
	if err != nil {
		wrapError(err, w)
		return
	}
	w.Header().Add(contentType, mimeTextPlain)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(out)
}
