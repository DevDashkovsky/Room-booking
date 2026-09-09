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

	if w.Code != http.StatusOK {
		t.Errorf("/_info status = %d, want 200", w.Code)
	}
}

func TestRouter_Readiness(t *testing.T) {
	for _, healthy := range []bool{false, true} {
		router := NewRouter(Deps{AuthH: NewAuthHandler("secret", nil), Ping: func(context.Context) error {
			if healthy {
				return nil
			}
			return errors.New("db down")
		}})
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/ready", nil))
		want := 503
		if healthy {
			want = 200
		}
		if w.Code != want {
			t.Errorf("healthy=%v status=%d want=%d", healthy, w.Code, want)
		}
	}
}
