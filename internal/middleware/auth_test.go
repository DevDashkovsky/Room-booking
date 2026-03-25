package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	myjwt "github.com/DevDashkovsky/room-booking/internal/jwt"
)

const testSecret = "test-secret"

func TestAuth_ValidToken(t *testing.T) {
	token, _ := myjwt.GenerateToken("user-1", "admin", testSecret)

	handler := Auth(testSecret)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		uid := GetUserID(r.Context())
		role := GetRole(r.Context())
		if uid != "user-1" {
			t.Errorf("userID = %q", uid)
		}
		if role != "admin" {
			t.Errorf("role = %q", role)
		}
		w.WriteHeader(http.StatusOK)
	}))

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d", w.Code)
	}
}

func TestAuth_MissingHeader(t *testing.T) {
	handler := Auth(testSecret)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not reach handler")
	}))

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d", w.Code)
	}

	var body map[string]map[string]string
	json.NewDecoder(w.Body).Decode(&body)
	if body["error"]["code"] != "UNAUTHORIZED" {
		t.Errorf("code = %q", body["error"]["code"])
	}
}

func TestAuth_InvalidToken(t *testing.T) {
	handler := Auth(testSecret)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not reach handler")
	}))

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Authorization", "Bearer garbage")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d", w.Code)
	}
}

func TestGetUserID_Empty(t *testing.T) {
	ctx := context.Background()
	if uid := GetUserID(ctx); uid != "" {
		t.Errorf("expected empty, got %q", uid)
	}
}

func TestGetRole_Empty(t *testing.T) {
	ctx := context.Background()
	if role := GetRole(ctx); role != "" {
		t.Errorf("expected empty, got %q", role)
	}
}
