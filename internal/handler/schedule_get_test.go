package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DevDashkovsky/room-booking/internal/domain"
)

type readScheduleService struct {
	calls int
	err   error
}

func (s *readScheduleService) Create(context.Context, *domain.Schedule) error { return nil }
func (s *readScheduleService) Update(context.Context, *domain.Schedule) error { return nil }
func (s *readScheduleService) Get(_ context.Context, id string) (*domain.Schedule, error) {
	s.calls++
	return &domain.Schedule{ID: "00000000-0000-0000-0000-000000000003", RoomID: id, DaysOfWeek: []int{1, 2}, StartTime: "09:00", EndTime: "18:00"}, s.err
}

func TestGetScheduleHTTP(t *testing.T) {
	const id = "00000000-0000-0000-0000-000000000004"
	tests := []struct {
		name, token, id string
		err             error
		status          int
		code            string
		calls           int
	}{
		{"admin", adminToken(), id, nil, 200, "", 1},
		{"user", userToken(), id, nil, 200, "", 1},
		{"unauthenticated", "", id, nil, 401, "UNAUTHORIZED", 0},
		{"invalid UUID", adminToken(), "bad", nil, 400, "INVALID_REQUEST", 0},
		{"missing room", adminToken(), id, domain.ErrRoomNotFound, 404, "ROOM_NOT_FOUND", 1},
		{"missing schedule", adminToken(), id, domain.ErrScheduleNotFound, 404, "SCHEDULE_NOT_FOUND", 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &readScheduleService{err: tt.err}
			router := NewRouter(Deps{JWTSecret: "secret", AuthH: NewAuthHandler("secret", nil), ScheduleH: NewScheduleHandler(svc)})
			req := httptest.NewRequest(http.MethodGet, "/rooms/"+tt.id+"/schedule", nil)
			if tt.token != "" {
				req.Header.Set("Authorization", "Bearer "+tt.token)
			}
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			if w.Code != tt.status || svc.calls != tt.calls {
				t.Fatalf("status=%d calls=%d body=%s", w.Code, svc.calls, w.Body.String())
			}
			if w.Header().Get("Content-Type") != "application/json" {
				t.Fatal("missing JSON content type")
			}
			if tt.code != "" {
				var body errorBody
				if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
					t.Fatal(err)
				}
				if body.Error.Code != tt.code {
					t.Fatalf("code=%q", body.Error.Code)
				}
			} else {
				var body struct{ Schedule domain.Schedule }
				if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
					t.Fatal(err)
				}
				if body.Schedule.ID == "" || body.Schedule.RoomID != id || len(body.Schedule.DaysOfWeek) != 2 || body.Schedule.StartTime != "09:00" || body.Schedule.EndTime != "18:00" {
					t.Fatalf("bad response: %+v", body)
				}
			}
		})
	}
}
