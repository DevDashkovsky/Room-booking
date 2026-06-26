package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRouter_Info(t *testing.T) {
	r := NewRouter(Deps{
		JWTSecret: "secret",
		AuthH:     NewAuthHandler("secret", nil),
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/_info", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("/_info status = %d, want 200", w.Code)
	}
}

func TestRouter_Info_DBHealthy(t *testing.T) {
	r := NewRouter(Deps{
		JWTSecret: "secret",
		AuthH:     NewAuthHandler("secret", nil),
		Ping:      func(context.Context) error { return nil },
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/_info", nil))

	if w.Code != http.StatusOK {
		t.Errorf("/_info status = %d, want 200", w.Code)
	}
}

func TestRouter_Info_DBDown(t *testing.T) {
	r := NewRouter(Deps{
		JWTSecret: "secret",
		AuthH:     NewAuthHandler("secret", nil),
		Ping:      func(context.Context) error { return errors.New("db down") },
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/_info", nil))

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("/_info status = %d, want 503", w.Code)
	}
}
