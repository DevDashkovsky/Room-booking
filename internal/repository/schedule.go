package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/DevDashkovsky/room-booking/internal/domain"
)

type ScheduleRepository struct {
	pool *pgxpool.Pool
}

func NewScheduleRepository(pool *pgxpool.Pool) *ScheduleRepository {
	return &ScheduleRepository{pool: pool}
}

func (r *ScheduleRepository) Create(ctx context.Context, s *domain.Schedule) error {
	return r.pool.QueryRow(ctx,
		`INSERT INTO schedules (room_id, days_of_week, start_time, end_time)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id`,
		s.RoomID, s.DaysOfWeek, s.StartTime, s.EndTime,
	).Scan(&s.ID)
}

func (r *ScheduleRepository) GetByRoomID(ctx context.Context, roomID string) (*domain.Schedule, error) {
	var s domain.Schedule
	err := r.pool.QueryRow(ctx,
		`SELECT id, room_id, days_of_week, start_time, end_time
		 FROM schedules
		 WHERE room_id = $1`,
		roomID,
	).Scan(&s.ID, &s.RoomID, &s.DaysOfWeek, &s.StartTime, &s.EndTime)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *ScheduleRepository) Exists(ctx context.Context, roomID string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM schedules WHERE room_id = $1)`,
		roomID,
	).Scan(&exists)
	return exists, err
}
