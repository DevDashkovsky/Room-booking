package handler

import (
	"net/http"

	"github.com/DevDashkovsky/room-booking/internal/domain"
	"github.com/DevDashkovsky/room-booking/internal/middleware"
	"github.com/DevDashkovsky/room-booking/internal/service"
)

type RoomHandler struct {
	roomSvc *service.RoomService
}

func NewRoomHandler(roomSvc *service.RoomService) *RoomHandler {
	return &RoomHandler{roomSvc: roomSvc}
}

func (h *RoomHandler) Create(w http.ResponseWriter, r *http.Request) {
	role := middleware.GetRole(r.Context())
	if role != "admin" {
		respondError(w, http.StatusForbidden, "FORBIDDEN", "admin role required")
		return
	}

	var req struct {
		Name        string  `json:"name"`
		Description *string `json:"description"`
		Capacity    *int    `json:"capacity"`
	}
	if !readBodyJSON(w, r, &req) {
		return
	}

	room := &domain.Room{
		Name:        req.Name,
		Description: req.Description,
		Capacity:    req.Capacity,
	}

	if err := h.roomSvc.Create(r.Context(), room); err != nil {
		handleServiceError(w, err)
		return
	}

	respondJSON(w, http.StatusCreated, map[string]any{"room": room})
}

func (h *RoomHandler) List(w http.ResponseWriter, r *http.Request) {
	rooms, err := h.roomSvc.List(r.Context())
	if err != nil {
		handleServiceError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{"rooms": rooms})
}
