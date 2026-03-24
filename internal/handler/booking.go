package handler

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/DevDashkovsky/room-booking/internal/middleware"
	"github.com/DevDashkovsky/room-booking/internal/service"
)

type BookingHandler struct {
	bookingSvc *service.BookingService
}

func NewBookingHandler(bookingSvc *service.BookingService) *BookingHandler {
	return &BookingHandler{bookingSvc: bookingSvc}
}

func (h *BookingHandler) Create(w http.ResponseWriter, r *http.Request) {
	role := middleware.GetRole(r.Context())
	if role != "user" {
		respondError(w, http.StatusForbidden, "FORBIDDEN", "only user role can create bookings")
		return
	}

	var req struct {
		SlotID               string `json:"slotId"`
		CreateConferenceLink bool   `json:"createConferenceLink"`
	}
	if !readBodyJSON(w, r, &req) {
		return
	}

	if req.SlotID == "" {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "slotId is required")
		return
	}

	userID := middleware.GetUserID(r.Context())

	booking, err := h.bookingSvc.Create(r.Context(), req.SlotID, userID, req.CreateConferenceLink)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	respondJSON(w, http.StatusCreated, map[string]any{"booking": booking})
}

func (h *BookingHandler) ListAll(w http.ResponseWriter, r *http.Request) {
	role := middleware.GetRole(r.Context())
	if role != "admin" {
		respondError(w, http.StatusForbidden, "FORBIDDEN", "admin role required")
		return
	}

	page, pageSize := parsePagination(r)

	result, err := h.bookingSvc.ListAll(r.Context(), page, pageSize)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"bookings":   result.Bookings,
		"pagination": result.Pagination,
	})
}

func (h *BookingHandler) ListMy(w http.ResponseWriter, r *http.Request) {
	role := middleware.GetRole(r.Context())
	if role != "user" {
		respondError(w, http.StatusForbidden, "FORBIDDEN", "only user role can view own bookings")
		return
	}

	userID := middleware.GetUserID(r.Context())

	bookings, err := h.bookingSvc.ListMy(r.Context(), userID)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{"bookings": bookings})
}

func (h *BookingHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	role := middleware.GetRole(r.Context())
	if role != "user" {
		respondError(w, http.StatusForbidden, "FORBIDDEN", "only user role can cancel bookings")
		return
	}

	bookingID := chi.URLParam(r, "bookingId")
	if bookingID == "" {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "missing bookingId")
		return
	}

	userID := middleware.GetUserID(r.Context())

	booking, err := h.bookingSvc.Cancel(r.Context(), bookingID, userID)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{"booking": booking})
}

func parsePagination(r *http.Request) (page, pageSize int) {
	page = 1
	pageSize = 20

	if v := r.URL.Query().Get("page"); v != "" {
		if p, err := strconv.Atoi(v); err == nil && p >= 1 {
			page = p
		}
	}

	if v := r.URL.Query().Get("pageSize"); v != "" {
		if ps, err := strconv.Atoi(v); err == nil && ps >= 1 {
			pageSize = ps
		}
	}

	if pageSize > 100 {
		pageSize = 100
	}

	return page, pageSize
}
