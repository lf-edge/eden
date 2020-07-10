package server

import (
	"github.com/gorilla/mux"
	"github.com/lf-edge/eden/eserver/pkg/manager"
	"net/http"
)

type apiHandler struct {
	manager *manager.EServerManager
}

func (h *apiHandler) getFile(w http.ResponseWriter, r *http.Request) {
	u := mux.Vars(r)["filename"]
	filePath, err := h.manager.GetFilePath(u)
	if err != nil {
		wrapError(err, w)
		return
	}
	http.ServeFile(w, r, filePath)
}
