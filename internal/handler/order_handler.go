package handler

import (
	"net/http"
)

func HandleOrder(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
	_, _ = w.Write([]byte("not implemented"))
}
