package handlers

import "net/http"

func WriteObject(writer http.ResponseWriter, request *http.Request) {
	writer.Write([]byte("foo"))
}
