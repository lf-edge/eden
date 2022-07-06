package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/lf-edge/eden/sdn/api"
	log "github.com/sirupsen/logrus"
)

type agent struct{}

func (h *agent) init() error {
	return nil
}

func (h *agent) getNetModel(w http.ResponseWriter, r *http.Request) {
	model := api.NetworkModel{}
	w.Header().Add("Content-Type", "application/json")
	resp, err := json.Marshal(model)
	if err != nil {
		errMsg := fmt.Sprintf("failed to marshal network model to JSON: %v", err)
		log.Error(errMsg)
		http.Error(w, errMsg, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(resp)
}

func (h *agent) applyNetModel(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "PUT net-model")
}

func (h *agent) getNetConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/vnd.graphviz")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "GET net-config")
}

func (h *agent) getStatus(w http.ResponseWriter, r *http.Request) {
	status := api.Status{}
	w.Header().Add("Content-Type", "application/json")
	resp, err := json.Marshal(status)
	if err != nil {
		errMsg := fmt.Sprintf("failed to marshal SDN status to JSON: %v", err)
		log.Error(errMsg)
		http.Error(w, errMsg, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(resp)
}