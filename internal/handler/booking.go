package handler

import (
	"math"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/DevDashkovsky/room-booking/internal/domain"
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

	if !domain.ValidUUID(req.SlotID) {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "slotId must be a UUID")
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

	page, pageSize, ok := parsePagination(r)
	if !ok {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid pagination")
		return
	}

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
	if !domain.ValidUUID(bookingID) {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "bookingId must be a UUID")
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

func parsePagination(r *http.Request) (page, pageSize int, ok bool) {
	page, pageSize = 1, 20
	for key, dst := range map[string]*int{"page": &page, "pageSize": &pageSize} {
		values, exists := r.URL.Query()[key]
		if !exists {
			continue
		}
		if len(values) != 1 || values[0] == "" {
			return 0, 0, false
		}
		for _, c := range values[0] {
			if c < '0' || c > '9' {
				return 0, 0, false
			}
		}
		value, err := strconv.Atoi(values[0])
		if err != nil || value < 1 {
			return 0, 0, false
		}
		*dst = value
	}
	if pageSize > 100 || page-1 > math.MaxInt/pageSize {
		return 0, 0, false
	}
	return page, pageSize, true
}
