package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/DevDashkovsky/room-booking/internal/domain"
)

type readScheduleRepo struct {
	updateScheduleRepo
	result  *domain.Schedule
	readErr error
	roomID  string
}

func (r *readScheduleRepo) GetByRoomID(_ context.Context, id string) (*domain.Schedule, error) {
	r.roomID = id
	return r.result, r.readErr
}

type readRoomFinder struct {
	room *domain.Room
	err  error
}

func (r readRoomFinder) GetByID(context.Context, string) (*domain.Room, error) { return r.room, r.err }

func TestScheduleServiceGet(t *testing.T) {
	const id = "abcdefab-0000-0000-0000-000000000001"
	failure := errors.New("db down")
	schedule := &domain.Schedule{ID: "schedule", RoomID: id, DaysOfWeek: []int{1}, StartTime: "09:00", EndTime: "18:00"}
	tests := []struct {
		name, id               string
		room                   *domain.Room
		roomErr                error
		schedule               *domain.Schedule
		scheduleErr, errorWant error
	}{
		{"found", strings.ToUpper(id), &domain.Room{ID: id}, nil, schedule, nil, nil},
		{"invalid ID", "bad", nil, nil, nil, nil, domain.ErrInvalidRequest},
		{"missing room", id, nil, nil, nil, nil, domain.ErrRoomNotFound},
		{"missing schedule", id, &domain.Room{ID: id}, nil, nil, nil, domain.ErrScheduleNotFound},
		{"room error", id, nil, failure, nil, nil, failure},
		{"schedule error", id, &domain.Room{ID: id}, nil, nil, failure, failure},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &readScheduleRepo{result: tt.schedule, readErr: tt.scheduleErr}
			svc := NewScheduleService(repo, readRoomFinder{room: tt.room, err: tt.roomErr})
			got, err := svc.Get(t.Context(), tt.id)
			if !errors.Is(err, tt.errorWant) {
				t.Fatalf("error=%v want=%v", err, tt.errorWant)
			}
			if err == nil && (got != schedule || repo.roomID != id) {
				t.Fatalf("result=%+v canonical room=%q", got, repo.roomID)
			}
			if (tt.room == nil || tt.roomErr != nil) && repo.roomID != "" {
				t.Fatal("read schedule before finding room")
			}
		})
	}
}
