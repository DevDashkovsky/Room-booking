package service

import (
	"context"
	"errors"
	"testing"

	"github.com/DevDashkovsky/room-booking/internal/domain"
)

func stringPointer(value string) *string { return &value }

func TestRoomService_Create_EmptyName(t *testing.T) {
	svc := &RoomService{}
	err := svc.Create(context.Background(), &domain.Room{Name: ""})
	if !errors.Is(err, domain.ErrInvalidRequest) {
		t.Errorf("expected ErrInvalidRequest, got %v", err)
	}
}

func TestRoomService_RejectsCapacityOverflow(t *testing.T) {
	for _, capacity := range []int{2147483648, -2147483649} {
		svc := NewRoomService(nil)
		if err := svc.Create(context.Background(), &domain.Room{Name: "room", Capacity: &capacity}); !errors.Is(err, domain.ErrInvalidRequest) {
			t.Fatalf("capacity %d: %v", capacity, err)
		}
	}
}

func TestRoomService_RejectsNullBytes(t *testing.T) {
	for _, room := range []*domain.Room{
		{Name: "room\x00name"},
		{Name: "room", Description: stringPointer("description\x00")},
	} {
		svc := NewRoomService(nil)
		if err := svc.Create(context.Background(), room); !errors.Is(err, domain.ErrInvalidRequest) {
			t.Fatalf("accepted PostgreSQL-incompatible text: %v", err)
		}
	}
}
