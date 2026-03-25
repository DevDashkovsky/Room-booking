package handler

import (
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/DevDashkovsky/room-booking/internal/domain"
)

func TestHandleServiceError(t *testing.T) {
	tests := []struct {
		err        error
		wantStatus int
		wantCode   string
	}{
		{domain.ErrInvalidRequest, 400, "INVALID_REQUEST"},
		{domain.ErrRoomNotFound, 404, "ROOM_NOT_FOUND"},
		{domain.ErrSlotNotFound, 404, "SLOT_NOT_FOUND"},
		{domain.ErrBookingNotFound, 404, "BOOKING_NOT_FOUND"},
		{domain.ErrScheduleExists, 409, "SCHEDULE_EXISTS"},
		{domain.ErrSlotAlreadyBooked, 409, "SLOT_ALREADY_BOOKED"},
		{domain.ErrForbidden, 403, "FORBIDDEN"},
		{errors.New("unknown"), 500, "INTERNAL_ERROR"},
	}

	for _, tt := range tests {
		w := httptest.NewRecorder()
		handleServiceError(w, tt.err)
		if w.Code != tt.wantStatus {
			t.Errorf("err=%v: status=%d, want %d", tt.err, w.Code, tt.wantStatus)
		}
	}
}
