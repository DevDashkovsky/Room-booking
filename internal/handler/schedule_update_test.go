package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DevDashkovsky/room-booking/internal/domain"
	"github.com/DevDashkovsky/room-booking/internal/service"
)

type scheduleUpdateRepository struct {
	calls int
	err   error
}

func (r *scheduleUpdateRepository) Create(context.Context, *domain.Schedule) error { return nil }
func (r *scheduleUpdateRepository) Update(_ context.Context, s *domain.Schedule) error {
	r.calls++
	s.ID = "00000000-0000-0000-0000-000000000003"
	return r.err
}

func TestScheduleUpdateHTTPContract(t *testing.T) {
	const valid = `{"daysOfWeek":[1,2],"startTime":"09:00","endTime":"11:00"}`
	const roomID = "00000000-0000-0000-0000-000000000004"
	tests := []struct {
		name, token, id, body string
		err                   error
		status                int
		code                  string
		calls                 int
	}{
		{"success", adminToken(), roomID, valid, nil, 200, "", 1},
		{"no auth", "", roomID, valid, nil, 401, "UNAUTHORIZED", 0},
		{"user", userToken(), roomID, valid, nil, 403, "FORBIDDEN", 0},
		{"bad ID", adminToken(), "bad", valid, nil, 400, "INVALID_REQUEST", 0},
		{"invalid body", adminToken(), roomID, `{}`, nil, 400, "INVALID_REQUEST", 0},
		{"unknown field", adminToken(), roomID, `{"extra":true}`, nil, 400, "INVALID_REQUEST", 0},
		{"missing room", adminToken(), roomID, valid, domain.ErrRoomNotFound, 404, "ROOM_NOT_FOUND", 1},
		{"missing schedule", adminToken(), roomID, valid, domain.ErrScheduleNotFound, 404, "SCHEDULE_NOT_FOUND", 1},
		{"conflict", adminToken(), roomID, valid, domain.ErrScheduleBookingConflict, 409, "SCHEDULE_BOOKING_CONFLICT", 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &scheduleUpdateRepository{err: tt.err}
			router := NewRouter(Deps{JWTSecret: "secret", AuthH: NewAuthHandler("secret", nil), ScheduleH: NewScheduleHandler(service.NewScheduleService(repo, nil))})
			req := httptest.NewRequest(http.MethodPost, "/rooms/"+tt.id+"/schedule/update", strings.NewReader(tt.body))
			if tt.token != "" {
				req.Header.Set("Authorization", "Bearer "+tt.token)
			}
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			if w.Code != tt.status || repo.calls != tt.calls {
				t.Fatalf("status=%d calls=%d body=%s", w.Code, repo.calls, w.Body.String())
			}
			if w.Header().Get("Content-Type") != "application/json" {
				t.Fatal("not JSON")
			}
			if tt.code != "" {
				var body errorBody
				if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
					t.Fatal(err)
				}
				if body.Error.Code != tt.code {
					t.Fatalf("code=%s", body.Error.Code)
				}
			} else {
				var body struct{ Schedule domain.Schedule }
				if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
					t.Fatal(err)
				}
				if body.Schedule.RoomID != roomID || body.Schedule.ID == "" || body.Schedule.StartTime != "09:00" {
					t.Fatalf("schedule response: %+v", body)
				}
			}
		})
	}
}

func (r *scheduleUpdateRepository) GetByRoomID(context.Context, string) (*domain.Schedule, error) {
	return nil, nil
}
