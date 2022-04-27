package handlers

import (
	log "github.com/sirupsen/logrus"
	"net/http"
)

func (h *ObjectHandler) WriteObject(writer http.ResponseWriter, request *http.Request) {
	id := getIdFromRequest(request)
	serverIp := h.backend.FindServerIPById(id)
	err := h.backend.Servers[serverIp].PutObject(request.Context(), id, request.Body, request.ContentLength)
	if err != nil {
		log.Error(err)
		http.Error(writer, "Server error", http.StatusInternalServerError)
		return
	}
	writer.WriteHeader(http.StatusOK)
}
