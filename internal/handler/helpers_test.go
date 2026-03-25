package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRespondJSON(t *testing.T) {
	w := httptest.NewRecorder()
	respondJSON(w, http.StatusOK, map[string]string{"key": "value"})

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}

	var body map[string]string
	json.NewDecoder(w.Body).Decode(&body)
	if body["key"] != "value" {
		t.Errorf("body = %v", body)
	}
}

func TestRespondError(t *testing.T) {
	w := httptest.NewRecorder()
	respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "bad")

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d", w.Code)
	}

	var body errorBody
	json.NewDecoder(w.Body).Decode(&body)
	if body.Error.Code != "INVALID_REQUEST" {
		t.Errorf("code = %q", body.Error.Code)
	}
	if body.Error.Message != "bad" {
		t.Errorf("message = %q", body.Error.Message)
	}
}

func TestReadBodyJSON_Valid(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"test"}`))
	w := httptest.NewRecorder()

	var dst struct {
		Name string `json:"name"`
	}
	ok := readBodyJSON(w, r, &dst)
	if !ok {
		t.Error("expected ok=true")
	}
	if dst.Name != "test" {
		t.Errorf("name = %q", dst.Name)
	}
}

func TestReadBodyJSON_Invalid(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`not json`))
	w := httptest.NewRecorder()

	var dst struct{}
	ok := readBodyJSON(w, r, &dst)
	if ok {
		t.Error("expected ok=false for invalid JSON")
	}
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d", w.Code)
	}
}
