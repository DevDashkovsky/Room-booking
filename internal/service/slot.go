package service

import (
	"context"
	"fmt"
	"time"

	"github.com/DevDashkovsky/room-booking/internal/domain"
	"github.com/DevDashkovsky/room-booking/internal/repository"
)

type SlotService struct {
	slotRepo     *repository.SlotRepository
	roomRepo     *repository.RoomRepository
	scheduleRepo *repository.ScheduleRepository
}

func NewSlotService(
	slotRepo *repository.SlotRepository,
	roomRepo *repository.RoomRepository,
	scheduleRepo *repository.ScheduleRepository,
) *SlotService {
	return &SlotService{
		slotRepo:     slotRepo,
		roomRepo:     roomRepo,
		scheduleRepo: scheduleRepo,
	}
}

func (s *SlotService) List(ctx context.Context, roomID string, date time.Time) ([]domain.Slot, error) {
	room, err := s.roomRepo.GetByID(ctx, roomID)
	if err != nil {
		return nil, fmt.Errorf("get room: %w", err)
	}
	if room == nil {
		return nil, domain.ErrRoomNotFound
	}

	schedule, err := s.scheduleRepo.GetByRoomID(ctx, roomID)
	if err != nil {
		return nil, fmt.Errorf("get schedule: %w", err)
	}
	if schedule == nil {
		return []domain.Slot{}, nil
	}

	slots, err := s.slotRepo.ListByRoomAndDate(ctx, roomID, date)
	if err != nil {
		return nil, fmt.Errorf("list slots: %w", err)
	}

	return slots, nil
}
