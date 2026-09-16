package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DevDashkovsky/room-booking/internal/domain"
)

type slotTestRoomFinder struct {
	room *domain.Room
}

func (f slotTestRoomFinder) GetByID(context.Context, string) (*domain.Room, error) {
	return f.room, nil
}

func TestSlotService_DateBounds(t *testing.T) {
	svc := NewSlotService(nil, slotTestRoomFinder{room: &domain.Room{ID: "room"}}, nil)

	past, err := svc.List(context.Background(), "unused", time.Now().UTC().AddDate(0, 0, -1))
	if err != nil || len(past) != 0 {
		t.Fatalf("past date = %v, %v; want empty list", past, err)
	}

	_, err = svc.List(context.Background(), "unused", time.Now().UTC().AddDate(0, 0, maxSlotHorizonDays+1))
	if !errors.Is(err, domain.ErrInvalidRequest) {
		t.Fatalf("far future date error = %v, want ErrInvalidRequest", err)
	}
}
