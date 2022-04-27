package handlers

import "github.com/spacelift-io/homework-object-storage/backend"

type ObjectHandler struct {
	backend *backend.Pool
}

func NewObjectHandler(backend *backend.Pool) *ObjectHandler {
	handler := ObjectHandler{}
	handler.backend = backend
	return &handler
}
