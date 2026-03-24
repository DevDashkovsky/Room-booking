package service

import (
	"context"
	"fmt"
	"time"

	"github.com/DevDashkovsky/room-booking/internal/domain"
	"github.com/DevDashkovsky/room-booking/internal/repository"
)

type BookingService struct {
	bookingRepo *repository.BookingRepository
	slotRepo    *repository.SlotRepository
}

func NewBookingService(bookingRepo *repository.BookingRepository, slotRepo *repository.SlotRepository) *BookingService {
	return &BookingService{
		bookingRepo: bookingRepo,
		slotRepo:    slotRepo,
	}
}

func (s *BookingService) Create(ctx context.Context, slotID, userID string, createConferenceLink bool) (*domain.Booking, error) {
	slot, err := s.slotRepo.GetByID(ctx, slotID)
	if err != nil {
		return nil, fmt.Errorf("get slot: %w", err)
	}
	if slot == nil {
		return nil, domain.ErrSlotNotFound
	}

	if slot.Start.Before(time.Now().UTC()) {
		return nil, domain.ErrInvalidRequest
	}

	existing, err := s.bookingRepo.ActiveBySlotID(ctx, slotID)
	if err != nil {
		return nil, fmt.Errorf("check active booking: %w", err)
	}
	if existing != nil {
		return nil, domain.ErrSlotAlreadyBooked
	}

	b := &domain.Booking{
		SlotID: slotID,
		UserID: userID,
	}

	if createConferenceLink {
		link := fmt.Sprintf("https://room-booking/meet/%s", slotID)
		b.ConferenceLink = &link
	}

	if err := s.bookingRepo.Create(ctx, b); err != nil {
		return nil, fmt.Errorf("create booking: %w", err)
	}

	return b, nil
}

type ListAllResult struct {
	Bookings   []domain.Booking
	Pagination domain.Pagination
}

func (s *BookingService) ListAll(ctx context.Context, page, pageSize int) (*ListAllResult, error) {
	total, err := s.bookingRepo.Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("count bookings: %w", err)
	}

	offset := (page - 1) * pageSize
	bookings, err := s.bookingRepo.ListAll(ctx, pageSize, offset)
	if err != nil {
		return nil, fmt.Errorf("list bookings: %w", err)
	}

	return &ListAllResult{
		Bookings: bookings,
		Pagination: domain.Pagination{
			Page:     page,
			PageSize: pageSize,
			Total:    total,
		},
	}, nil
}

func (s *BookingService) ListMy(ctx context.Context, userID string) ([]domain.Booking, error) {
	bookings, err := s.bookingRepo.ListByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list user bookings: %w", err)
	}
	return bookings, nil
}

func (s *BookingService) Cancel(ctx context.Context, bookingID, userID string) (*domain.Booking, error) {
	booking, err := s.bookingRepo.GetByID(ctx, bookingID)
	if err != nil {
		return nil, fmt.Errorf("get booking: %w", err)
	}
	if booking == nil {
		return nil, domain.ErrBookingNotFound
	}

	if booking.UserID != userID {
		return nil, domain.ErrForbidden
	}

	if booking.Status == "cancelled" {
		return booking, nil
	}

	if err := s.bookingRepo.Cancel(ctx, bookingID); err != nil {
		return nil, fmt.Errorf("cancel booking: %w", err)
	}

	booking.Status = "cancelled"
	return booking, nil
}
