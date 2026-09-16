package service

import (
	"context"
	"fmt"
	"math"
	"strings"

	"github.com/DevDashkovsky/room-booking/internal/domain"
)

type roomRepository interface {
	Create(context.Context, *domain.Room) error
	List(context.Context) ([]domain.Room, error)
}

type RoomService struct {
	roomRepo roomRepository
}

func NewRoomService(roomRepo roomRepository) *RoomService {
	return &RoomService{roomRepo: roomRepo}
}

func (s *RoomService) Create(ctx context.Context, room *domain.Room) error {
	if room.Name == "" || strings.ContainsRune(room.Name, '\x00') ||
		(room.Description != nil && strings.ContainsRune(*room.Description, '\x00')) ||
		(room.Capacity != nil && (*room.Capacity < math.MinInt32 || *room.Capacity > math.MaxInt32)) {
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
