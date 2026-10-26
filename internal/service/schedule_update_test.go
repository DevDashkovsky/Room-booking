package service

import (
	"context"
	"errors"
	"testing"

	"github.com/DevDashkovsky/room-booking/internal/domain"
)

type updateScheduleRepo struct {
	calls int
	err   error
}

func (r *updateScheduleRepo) Create(context.Context, *domain.Schedule) error { return nil }
func (r *updateScheduleRepo) Update(_ context.Context, s *domain.Schedule) error {
	r.calls++
	s.ID = "existing"
	return r.err
}

func TestScheduleServiceUpdate(t *testing.T) {
	repo := &updateScheduleRepo{}
	svc := NewScheduleService(repo, nil)
	if err := svc.Update(t.Context(), &domain.Schedule{}); !errors.Is(err, domain.ErrInvalidRequest) || repo.calls != 0 {
		t.Fatalf("invalid schedule: %v calls=%d", err, repo.calls)
	}
	for _, want := range []error{nil, domain.ErrScheduleNotFound, domain.ErrScheduleBookingConflict} {
		repo.err = want
		schedule := &domain.Schedule{RoomID: "room", DaysOfWeek: []int{1}, StartTime: "09:00", EndTime: "10:00"}
		err := svc.Update(t.Context(), schedule)
		if !errors.Is(err, want) {
			t.Fatalf("error=%v want=%v", err, want)
		}
		if want == nil && schedule.ID != "existing" {
			t.Fatal("missing schedule ID")
		}
	}
}

func (r *updateScheduleRepo) GetByRoomID(context.Context, string) (*domain.Schedule, error) {
	return nil, nil
}
