package service

import (
	"context"
	"fmt"
	"time"

	"github.com/DevDashkovsky/room-booking/internal/domain"
)

type slotRepository interface {
	ListAvailable(context.Context, string, time.Time) ([]domain.Slot, error)
}

type SlotService struct {
	slotRepo slotRepository
	roomRepo roomFinder
}

const maxSlotHorizonDays = 365

func NewSlotService(slotRepo slotRepository, roomRepo roomFinder) *SlotService {
	return &SlotService{slotRepo: slotRepo, roomRepo: roomRepo}
}

func (s *SlotService) List(ctx context.Context, roomID string, date time.Time) ([]domain.Slot, error) {
	today := utcDay(time.Now())
	requestedDay := utcDay(date)
	if requestedDay.After(today.AddDate(0, 0, maxSlotHorizonDays)) {
		return nil, domain.ErrInvalidRequest
	}

	room, err := s.roomRepo.GetByID(ctx, roomID)
	if err != nil {
		return nil, fmt.Errorf("get room: %w", err)
	}
	if room == nil {
		return nil, domain.ErrRoomNotFound
	}
	if requestedDay.Before(today) {
		return []domain.Slot{}, nil
	}

	slots, err := s.slotRepo.ListAvailable(ctx, roomID, requestedDay)
	if err != nil {
		return nil, fmt.Errorf("list slots: %w", err)
	}

	return slots, nil
}

func utcDay(value time.Time) time.Time {
	value = value.UTC()
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, time.UTC)
}
