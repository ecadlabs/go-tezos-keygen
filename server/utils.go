package server

import (
	"encoding/json"
	"net/http"
)

func jsonResponse(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func jsonError(w http.ResponseWriter, err error, status int) {
	type errorResponse struct {
		Error string `json:"error,omitempty"`
	}
	res := errorResponse{
		Error: err.Error(),
	}
	jsonResponse(w, status, &res)
}
