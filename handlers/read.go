package handlers

import (
	"fmt"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"github.com/spacelift-io/homework-object-storage/backend"
	"io"
	"net/http"
	"net/url"
)

type ReadObject struct {
	backend *backend.Pool
}

func NewReadObjectHandler(backend *backend.Pool) http.HandlerFunc {
	read := ReadObject{}
	read.backend = backend
	return read.ReadObject
}

func (r *ReadObject) ReadObject(writer http.ResponseWriter, request *http.Request) {
	id := getIdFromRequest(request)

	server, err := r.backend.FindObjectServer(request.Context(), id)
	if err != nil {
		//Triggers only when issues with backend
		log.Error("problem with accessing backend servers", err)
		http.Error(writer, "Server error", http.StatusInternalServerError)
		return
	}

	if server == nil {
		//No server found for the object = 404
		log.Debugf("Object with ID: %s does not found on any server", id)
		http.NotFound(writer, request)
		return
	}

	url, err := server.PresignedGetObject(request.Context(), id)
	if err != nil {
		log.Error(fmt.Errorf("error generating presigned object url for id: %s", id), err)
		http.Error(writer, "Server error", http.StatusInternalServerError)
		return
	}

	proxyUrl(writer, url)
}

func proxyUrl(writer http.ResponseWriter, url *url.URL) {
	client := &http.Client{}
	resp, err := client.Get(url.String())
	defer resp.Body.Close()
	if err != nil {
		http.Error(writer, "Server error", http.StatusInternalServerError)
		log.Error("error when proxy attempt", err)
		return
	}
	writer.WriteHeader(resp.StatusCode)
	io.Copy(writer, resp.Body)
}

func getIdFromRequest(request *http.Request) string {
	vars := mux.Vars(request)
	return vars["id"]
}
