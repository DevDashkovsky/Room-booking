package handler

import (
	"net/http"

	"github.com/DevDashkovsky/room-booking/internal/jwt"
)

const (
	adminUserID = "00000000-0000-0000-0000-000000000001"
	regularUserID = "00000000-0000-0000-0000-000000000002"
)

type AuthHandler struct {
	jwtSecret string
}

func NewAuthHandler(secret string) *AuthHandler {
	return &AuthHandler{jwtSecret: secret}
}

func (h *AuthHandler) DummyLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Role string `json:"role"`
	}
	if !readBodyJSON(w, r, &req) {
		return
	}

	var userID string
	switch req.Role {
	case "admin":
		userID = adminUserID
	case "user":
		userID = regularUserID
	default:
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "role must be 'admin' or 'user'")
		return
	}

	token, err := jwt.GenerateToken(userID, req.Role, h.jwtSecret)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to generate token")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"token": token})
}
