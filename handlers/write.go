package handlers

import (
	log "github.com/sirupsen/logrus"
	"github.com/spacelift-io/homework-object-storage/backend"
	"net/http"
)

type WriteObject struct {
	backend *backend.Backend
}

func NewWriteObjectHandler(backend *backend.Backend) http.HandlerFunc {
	read := WriteObject{}
	read.backend = backend
	return read.WriteObject
}

func (w *WriteObject) WriteObject(writer http.ResponseWriter, request *http.Request) {
	id := getIdFromRequest(request)
	serverIp := w.backend.FindServerIPById(id)
	err := w.backend.Servers[serverIp].PutObject(request.Context(), id, request.Body, request.ContentLength)
	if err != nil {
		log.Error(err)
		http.Error(writer, "Server error", http.StatusInternalServerError)
		return
	}
}
