package handler

import (
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
