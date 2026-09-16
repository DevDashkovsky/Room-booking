package handler

import (
	"context"
	"encoding/json"
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
	if w.Header().Get("X-Request-Id") == "" {
		t.Error("/_info response is missing X-Request-Id")
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

func TestRouter_JSONErrorsForUnknownRouteAndMethod(t *testing.T) {
	router := NewRouter(Deps{AuthH: NewAuthHandler("secret", nil)})
	tests := []struct {
		method     string
		path       string
		wantStatus int
		wantCode   string
	}{
		{http.MethodGet, "/unknown", http.StatusNotFound, "NOT_FOUND"},
		{http.MethodGet, "/dummyLogin", http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED"},
	}

	for _, tt := range tests {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest(tt.method, tt.path, nil))
		if w.Code != tt.wantStatus {
			t.Fatalf("%s %s: status=%d, want=%d", tt.method, tt.path, w.Code, tt.wantStatus)
		}
		if contentType := w.Header().Get("Content-Type"); contentType != "application/json" {
			t.Fatalf("%s %s: content type=%q", tt.method, tt.path, contentType)
		}
		var body errorBody
		if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.Error.Code != tt.wantCode {
			t.Fatalf("%s %s: code=%q, want=%q", tt.method, tt.path, body.Error.Code, tt.wantCode)
		}
	}
}

func TestObserveRequests_RecoversPanicAsJSON(t *testing.T) {
	handler := observeRequests(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("boom")
	}))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/panic", nil))

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d, want=%d", w.Code, http.StatusInternalServerError)
	}
	var body errorBody
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Error.Code != "INTERNAL_ERROR" {
		t.Fatalf("code=%q, want INTERNAL_ERROR", body.Error.Code)
	}
}
