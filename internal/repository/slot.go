package repository

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/DevDashkovsky/room-booking/internal/domain"
)

type SlotRepository struct {
	pool *pgxpool.Pool
}

func NewSlotRepository(pool *pgxpool.Pool) *SlotRepository {
	return &SlotRepository{pool: pool}
}

func (r *SlotRepository) BulkCreate(ctx context.Context, slots []domain.Slot) error {
	if len(slots) == 0 {
		return nil
	}

	_, err := r.pool.CopyFrom(
		ctx,
		pgx.Identifier{"slots"},
		[]string{"room_id", "start_at", "end_at"},
		pgx.CopyFromSlice(len(slots), func(i int) ([]any, error) {
			return []any{slots[i].RoomID, slots[i].Start, slots[i].End}, nil
		}),
	)
	return err
}

func (r *SlotRepository) ListByRoomAndDate(ctx context.Context, roomID string, date time.Time) ([]domain.Slot, error) {
	dayStart := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
	dayEnd := dayStart.Add(24 * time.Hour)

	rows, err := r.pool.Query(ctx,
		`SELECT s.id, s.room_id, s.start_at, s.end_at
		 FROM slots s
		 LEFT JOIN bookings b ON b.slot_id = s.id AND b.status = 'active'
		 WHERE s.room_id = $1 AND s.start_at >= $2 AND s.start_at < $3 AND b.id IS NULL
		 ORDER BY s.start_at`,
		roomID, dayStart, dayEnd,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	slots := make([]domain.Slot, 0)
	for rows.Next() {
		var slot domain.Slot
		if err := rows.Scan(&slot.ID, &slot.RoomID, &slot.Start, &slot.End); err != nil {
			return nil, err
		}
		slots = append(slots, slot)
	}
	return slots, rows.Err()
}

func (r *SlotRepository) GetByID(ctx context.Context, id string) (*domain.Slot, error) {
	var s domain.Slot
	err := r.pool.QueryRow(ctx,
		`SELECT id, room_id, start_at, end_at FROM slots WHERE id = $1`,
		id,
	).Scan(&s.ID, &s.RoomID, &s.Start, &s.End)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}
