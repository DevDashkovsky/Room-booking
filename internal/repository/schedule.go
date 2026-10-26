package repository

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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
	err := r.pool.QueryRow(ctx,
		`INSERT INTO schedules (room_id, days_of_week, start_time, end_time)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id`,
		s.RoomID, s.DaysOfWeek, s.StartTime, s.EndTime,
	).Scan(&s.ID)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation && pgErr.ConstraintName == "schedules_room_id_key" {
		return domain.ErrScheduleExists
	}
	return err
}

type rowQuerier interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func (r *ScheduleRepository) GetByRoomID(ctx context.Context, roomID string) (*domain.Schedule, error) {
	return getSchedule(ctx, r.pool, roomID)
}

func getSchedule(ctx context.Context, q rowQuerier, roomID string) (*domain.Schedule, error) {
	var s domain.Schedule
	err := q.QueryRow(ctx,
		`SELECT id, room_id, days_of_week, to_char(start_time, 'HH24:MI'), to_char(end_time, 'HH24:MI')
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

// Room locks serialize schedule changes with slot generation and booking writes.
func lockRoom(ctx context.Context, tx pgx.Tx, roomID string, exclusive bool) (string, error) {
	query := "SELECT id FROM rooms WHERE id=$1 FOR SHARE"
	if exclusive {
		query = "SELECT id FROM rooms WHERE id=$1 FOR UPDATE"
	}
	var canonicalID string
	err := tx.QueryRow(ctx, query, roomID).Scan(&canonicalID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", domain.ErrRoomNotFound
	}
	return canonicalID, err
}

func (r *ScheduleRepository) Update(ctx context.Context, schedule *domain.Schedule) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	roomID, err := lockRoom(ctx, tx, schedule.RoomID, true)
	if err != nil {
		return err
	}
	schedule.RoomID = roomID
	existing, err := getSchedule(ctx, tx, roomID)
	if err != nil {
		return err
	}
	if existing == nil {
		return domain.ErrScheduleNotFound
	}
	var now time.Time
	if err := tx.QueryRow(ctx, "SELECT clock_timestamp()").Scan(&now); err != nil {
		return err
	}
	rows, err := tx.Query(ctx, `SELECT s.id,s.room_id,s.start_at,s.end_at
 FROM slots s JOIN bookings b ON b.slot_id=s.id
 WHERE s.room_id=$1 AND b.status='active' AND s.end_at>$2`, roomID, now)
	if err != nil {
		return err
	}
	for rows.Next() {
		var slot domain.Slot
		if err := rows.Scan(&slot.ID, &slot.RoomID, &slot.Start, &slot.End); err != nil {
			rows.Close()
			return err
		}
		if !schedule.Allows(&slot) {
			rows.Close()
			return domain.ErrScheduleBookingConflict
		}
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	if err := tx.QueryRow(ctx, `UPDATE schedules SET days_of_week=$2,start_time=$3,end_time=$4
 WHERE room_id=$1 RETURNING id,to_char(start_time,'HH24:MI'),to_char(end_time,'HH24:MI')`,
		roomID, schedule.DaysOfWeek, schedule.StartTime, schedule.EndTime).Scan(&schedule.ID, &schedule.StartTime, &schedule.EndTime); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
