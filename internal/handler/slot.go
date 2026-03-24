package handler

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/DevDashkovsky/room-booking/internal/service"
)

type SlotHandler struct {
	slotSvc *service.SlotService
}

func NewSlotHandler(slotSvc *service.SlotService) *SlotHandler {
	return &SlotHandler{slotSvc: slotSvc}
}

func (h *SlotHandler) List(w http.ResponseWriter, r *http.Request) {
	roomID := chi.URLParam(r, "roomId")
	if roomID == "" {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "missing roomId")
		return
	}

	dateStr := r.URL.Query().Get("date")
	if dateStr == "" {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "missing date parameter")
		return
	}

	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid date format, expected YYYY-MM-DD")
		return
	}

	slots, err := h.slotSvc.List(r.Context(), roomID, date)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{"slots": slots})
}
