package service

import (
	"context"
	"fmt"

	"github.com/DevDashkovsky/room-booking/internal/domain"
)

type scheduleRepository interface {
	Create(context.Context, *domain.Schedule) error
	Update(context.Context, *domain.Schedule) error
}

type roomFinder interface {
	GetByID(context.Context, string) (*domain.Room, error)
}

type ScheduleService struct {
	scheduleRepo scheduleRepository
	roomRepo     roomFinder
}

func NewScheduleService(
	scheduleRepo scheduleRepository,
	roomRepo roomFinder,
) *ScheduleService {
	return &ScheduleService{
		scheduleRepo: scheduleRepo,
		roomRepo:     roomRepo,
	}
}

func (s *ScheduleService) Create(ctx context.Context, schedule *domain.Schedule) error {
	if err := validateSchedule(schedule); err != nil {
		return err
	}

	room, err := s.roomRepo.GetByID(ctx, schedule.RoomID)
	if err != nil {
		return fmt.Errorf("get room: %w", err)
	}
	if room == nil {
		return domain.ErrRoomNotFound
	}

	if err := s.scheduleRepo.Create(ctx, schedule); err != nil {
		return fmt.Errorf("create schedule: %w", err)
	}

	return nil
}

func (s *ScheduleService) Update(ctx context.Context, schedule *domain.Schedule) error {
	if err := validateSchedule(schedule); err != nil {
		return err
	}
	if err := s.scheduleRepo.Update(ctx, schedule); err != nil {
		return fmt.Errorf("update schedule: %w", err)
	}
	return nil
}

func validateSchedule(s *domain.Schedule) error {
	if len(s.DaysOfWeek) == 0 {
		return domain.ErrInvalidRequest
	}
	for _, d := range s.DaysOfWeek {
		if d < 1 || d > 7 {
			return domain.ErrInvalidRequest
		}
	}

	startH, startM, ok1 := domain.ParseScheduleTime(s.StartTime)
	endH, endM, ok2 := domain.ParseScheduleTime(s.EndTime)
	if !ok1 || !ok2 {
		return domain.ErrInvalidRequest
	}

	if startH*60+startM >= endH*60+endM {
		return domain.ErrInvalidRequest
	}

	return nil
}
