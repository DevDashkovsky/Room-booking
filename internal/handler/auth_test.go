package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDummyLogin_Admin(t *testing.T) {
	h := NewAuthHandler("test-secret", nil)
	r := httptest.NewRequest(http.MethodPost, "/dummyLogin", strings.NewReader(`{"role":"admin"}`))
	w := httptest.NewRecorder()

	h.DummyLogin(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}

	var body map[string]string
	json.NewDecoder(w.Body).Decode(&body)
	if body["token"] == "" {
		t.Error("token is empty")
	}
}

func TestDummyLogin_User(t *testing.T) {
	h := NewAuthHandler("test-secret", nil)
	r := httptest.NewRequest(http.MethodPost, "/dummyLogin", strings.NewReader(`{"role":"user"}`))
	w := httptest.NewRecorder()

	h.DummyLogin(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
}

func TestDummyLogin_InvalidRole(t *testing.T) {
	h := NewAuthHandler("test-secret", nil)
	r := httptest.NewRequest(http.MethodPost, "/dummyLogin", strings.NewReader(`{"role":"superadmin"}`))
	w := httptest.NewRecorder()

	h.DummyLogin(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestDummyLogin_InvalidJSON(t *testing.T) {
	h := NewAuthHandler("test-secret", nil)
	r := httptest.NewRequest(http.MethodPost, "/dummyLogin", strings.NewReader(`not json`))
	w := httptest.NewRecorder()

	h.DummyLogin(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}
