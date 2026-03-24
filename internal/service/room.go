package service

import (
	"context"
	"fmt"

	"github.com/DevDashkovsky/room-booking/internal/domain"
	"github.com/DevDashkovsky/room-booking/internal/repository"
)

type RoomService struct {
	roomRepo *repository.RoomRepository
}

func NewRoomService(roomRepo *repository.RoomRepository) *RoomService {
	return &RoomService{roomRepo: roomRepo}
}

func (s *RoomService) Create(ctx context.Context, room *domain.Room) error {
	if room.Name == "" {
		return domain.ErrInvalidRequest
	}
	if err := s.roomRepo.Create(ctx, room); err != nil {
		return fmt.Errorf("create room: %w", err)
	}
	return nil
}

func (s *RoomService) List(ctx context.Context) ([]domain.Room, error) {
	rooms, err := s.roomRepo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list rooms: %w", err)
	}
	return rooms, nil
}
