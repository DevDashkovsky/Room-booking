package service

import (
	"context"
	"errors"
	"testing"

	"github.com/DevDashkovsky/room-booking/internal/domain"
)

type moveRepositoryStub struct {
	bookingRepository
	calls int
	err   error
}

func (r *moveRepositoryStub) Move(_ context.Context, _, _, _ string) (*domain.Booking, error) {
	r.calls++
	return nil, r.err
}

func TestBookingMoveValidatesIDsAndPreservesErrors(t *testing.T) {
	const valid = "00000000-0000-0000-0000-000000000002"
	for index := range 3 {
		repo := &moveRepositoryStub{}
		ids := []string{valid, valid, valid}
		ids[index] = "bad"
		_, err := NewBookingService(repo, nil).Move(t.Context(), ids[0], ids[1], ids[2])
		if !errors.Is(err, domain.ErrInvalidRequest) || repo.calls != 0 {
			t.Fatalf("invalid ID %d: %v, calls=%d", index, err, repo.calls)
		}
	}
	repo := &moveRepositoryStub{err: context.DeadlineExceeded}
	_, err := NewBookingService(repo, nil).Move(t.Context(), valid, valid, valid)
	if !errors.Is(err, context.DeadlineExceeded) || repo.calls != 1 {
		t.Fatalf("error lost: %v", err)
	}
}
