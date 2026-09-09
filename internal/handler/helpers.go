package handler

import (
	"encoding/json"
	"io"
	"net/http"
)

type errorBody struct {
	Error errorDetail `json:"error"`
}

type errorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func respondJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, status int, code, message string) {
	a := errorBody{Error: errorDetail{Code: code, Message: message}}
	respondJSON(w, status, a)
}

func readBodyJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	const maxBodyBytes = 1 << 20
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(dst); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid JSON or body exceeds 1 MiB")
		return false
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "expected one JSON value")
		return false
	}
	return true
}
