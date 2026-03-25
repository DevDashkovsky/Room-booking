package service

import (
	"context"
	"errors"
	"testing"

	"github.com/DevDashkovsky/room-booking/internal/domain"
)

func TestRoomService_Create_EmptyName(t *testing.T) {
	svc := &RoomService{}
	err := svc.Create(context.Background(), &domain.Room{Name: ""})
	if !errors.Is(err, domain.ErrInvalidRequest) {
		t.Errorf("expected ErrInvalidRequest, got %v", err)
	}
}
