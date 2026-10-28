package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DevDashkovsky/room-booking/internal/domain"
	"github.com/DevDashkovsky/room-booking/internal/middleware"
)

type moveServiceStub struct {
	bookingService
	err   error
	calls int
	owner string
}

func (s *moveServiceStub) Move(ctx context.Context, bookingID, slotID, userID string) (*domain.Booking, error) {
	s.calls++
	s.owner = userID
	if userID != middleware.GetUserID(ctx) {
		panic("owner not taken from request context")
	}
	return &domain.Booking{ID: bookingID, SlotID: slotID, UserID: userID, Status: "active"}, s.err
}

func TestBookingMoveHTTPContract(t *testing.T) {
	const id = "00000000-0000-0000-0000-000000000003"
	const slot = "00000000-0000-0000-0000-000000000004"
	const body = `{"slotId":"` + slot + `"}`
	for _, tt := range []struct {
		name, token, id, body string
		err                   error
		status                int
		code                  string
		calls                 int
	}{
		{"success", userToken(), id, body, nil, 200, "", 1},
		{"no auth", "", id, body, nil, 401, "UNAUTHORIZED", 0},
		{"admin", adminToken(), id, body, nil, 403, "FORBIDDEN", 0},
		{"bad booking ID", userToken(), "bad", body, nil, 400, "INVALID_REQUEST", 0},
		{"bad slot ID", userToken(), id, `{"slotId":"bad"}`, nil, 400, "INVALID_REQUEST", 0},
		{"missing slot", userToken(), id, `{}`, nil, 400, "INVALID_REQUEST", 0},
		{"null", userToken(), id, `null`, nil, 400, "INVALID_REQUEST", 0},
		{"owner override", userToken(), id, `{"slotId":"` + slot + `","userId":"someone"}`, nil, 400, "INVALID_REQUEST", 0},
		{"extra JSON", userToken(), id, body + `{}`, nil, 400, "INVALID_REQUEST", 0},
		{"foreign", userToken(), id, body, domain.ErrForbidden, 403, "FORBIDDEN", 1},
		{"booking not found", userToken(), id, body, domain.ErrBookingNotFound, 404, "BOOKING_NOT_FOUND", 1},
		{"slot not found", userToken(), id, body, domain.ErrSlotNotFound, 404, "SLOT_NOT_FOUND", 1},
		{"occupied", userToken(), id, body, domain.ErrSlotAlreadyBooked, 409, "SLOT_ALREADY_BOOKED", 1},
		{"unavailable", userToken(), id, body, domain.ErrBookingMoveConflict, 409, "BOOKING_MOVE_CONFLICT", 1},
		{"deadline", userToken(), id, body, context.DeadlineExceeded, 503, "UNAVAILABLE", 1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			stub := &moveServiceStub{err: tt.err}
			router := NewRouter(Deps{JWTSecret: "secret", AuthH: NewAuthHandler("secret", nil), BookingH: NewBookingHandler(stub)})
			req := httptest.NewRequest(http.MethodPost, "/bookings/"+tt.id+"/move", strings.NewReader(tt.body))
			if tt.token != "" {
				req.Header.Set("Authorization", "Bearer "+tt.token)
			}
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			if w.Code != tt.status || stub.calls != tt.calls {
				t.Fatalf("status=%d calls=%d body=%s", w.Code, stub.calls, w.Body.String())
			}
			if tt.code != "" {
				var result errorBody
				if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil || result.Error.Code != tt.code {
					t.Fatalf("error body=%s err=%v", w.Body.String(), err)
				}
			} else {
				var result struct{ Booking domain.Booking }
				if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil || result.Booking.ID != id || result.Booking.SlotID != slot || stub.owner == "" {
					t.Fatalf("booking body=%s err=%v", w.Body.String(), err)
				}
			}
		})
	}
}
