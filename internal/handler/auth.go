package handler

import (
	"net/http"

	"github.com/DevDashkovsky/room-booking/internal/jwt"
	"github.com/DevDashkovsky/room-booking/internal/service"
)

const (
	adminUserID   = "00000000-0000-0000-0000-000000000001"
	regularUserID = "00000000-0000-0000-0000-000000000002"
)

type AuthHandler struct {
	jwtSecret string
	authSvc   *service.AuthService
}

func NewAuthHandler(secret string, authSvc *service.AuthService) *AuthHandler {
	return &AuthHandler{jwtSecret: secret, authSvc: authSvc}
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

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Role     string `json:"role"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if !readBodyJSON(w, r, &req) {
		return
	}

	user, err := h.authSvc.Register(r.Context(), req.Email, req.Password, req.Role)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	respondJSON(w, http.StatusCreated, map[string]any{"user": user})
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if !readBodyJSON(w, r, &req) {
		return
	}

	token, err := h.authSvc.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"token": token})
}
