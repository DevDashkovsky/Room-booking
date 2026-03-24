package handler

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/DevDashkovsky/room-booking/internal/domain"
	"github.com/DevDashkovsky/room-booking/internal/middleware"
	"github.com/DevDashkovsky/room-booking/internal/service"
)

type ScheduleHandler struct {
	scheduleSvc *service.ScheduleService
}

func NewScheduleHandler(scheduleSvc *service.ScheduleService) *ScheduleHandler {
	return &ScheduleHandler{scheduleSvc: scheduleSvc}
}

func (h *ScheduleHandler) Create(w http.ResponseWriter, r *http.Request) {
	role := middleware.GetRole(r.Context())
	if role != "admin" {
		respondError(w, http.StatusForbidden, "FORBIDDEN", "admin role required")
		return
	}

	roomID := chi.URLParam(r, "roomId")
	if roomID == "" {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "missing roomId")
		return
	}

	var req struct {
		DaysOfWeek []int  `json:"daysOfWeek"`
		StartTime  string `json:"startTime"`
		EndTime    string `json:"endTime"`
	}
	if !readBodyJSON(w, r, &req) {
		return
	}

	schedule := &domain.Schedule{
		RoomID:     roomID,
		DaysOfWeek: req.DaysOfWeek,
		StartTime:  req.StartTime,
		EndTime:    req.EndTime,
	}

	err := h.scheduleSvc.Create(r.Context(), schedule)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	respondJSON(w, http.StatusCreated, map[string]any{"schedule": schedule})
}

func handleServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidRequest):
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request")
	case errors.Is(err, domain.ErrRoomNotFound):
		respondError(w, http.StatusNotFound, "ROOM_NOT_FOUND", "room not found")
	case errors.Is(err, domain.ErrSlotNotFound):
		respondError(w, http.StatusNotFound, "SLOT_NOT_FOUND", "slot not found")
	case errors.Is(err, domain.ErrBookingNotFound):
		respondError(w, http.StatusNotFound, "BOOKING_NOT_FOUND", "booking not found")
	case errors.Is(err, domain.ErrScheduleExists):
		respondError(w, http.StatusConflict, "SCHEDULE_EXISTS", "schedule for this room already exists and cannot be changed")
	case errors.Is(err, domain.ErrSlotAlreadyBooked):
		respondError(w, http.StatusConflict, "SLOT_ALREADY_BOOKED", "slot is already booked")
	case errors.Is(err, domain.ErrForbidden):
		respondError(w, http.StatusForbidden, "FORBIDDEN", "access denied")
	default:
		respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
	}
}
