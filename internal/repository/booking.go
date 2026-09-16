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

const pgUniqueViolation = "23505"

type BookingRepository struct {
	pool *pgxpool.Pool
}

func NewBookingRepository(pool *pgxpool.Pool) *BookingRepository {
	return &BookingRepository{pool: pool}
}

func (r *BookingRepository) Create(ctx context.Context, b *domain.Booking) error {
	err := r.pool.QueryRow(ctx,
		`INSERT INTO bookings (slot_id, user_id, conference_link)
		 SELECT s.id, $2, $3
		 FROM slots s
		 WHERE s.id = $1 AND s.start_at >= NOW()
		 RETURNING id, status, created_at`,
		b.SlotID, b.UserID, b.ConferenceLink,
	).Scan(&b.ID, &b.Status, &b.CreatedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrInvalidRequest
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation {
		return domain.ErrSlotAlreadyBooked
	}
	return err
}

func (r *BookingRepository) ListAllPage(ctx context.Context, limit, offset int) ([]domain.Booking, int, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var total int
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM bookings`).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := tx.Query(ctx,
		`SELECT id, slot_id, user_id, status, conference_link, created_at
		 FROM bookings
		 ORDER BY created_at DESC, id DESC
		 LIMIT $1 OFFSET $2`,
		limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}

	bookings := make([]domain.Booking, 0)
	for rows.Next() {
		var b domain.Booking
		if err := rows.Scan(&b.ID, &b.SlotID, &b.UserID, &b.Status, &b.ConferenceLink, &b.CreatedAt); err != nil {
			rows.Close()
			return nil, 0, err
		}
		bookings = append(bookings, b)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, 0, err
	}
	rows.Close()

	if err := tx.Commit(ctx); err != nil {
		return nil, 0, err
	}
	return bookings, total, nil
}

func (r *BookingRepository) GetByID(ctx context.Context, id string) (*domain.Booking, error) {
	var b domain.Booking
	err := r.pool.QueryRow(ctx,
		`SELECT id, slot_id, user_id, status, conference_link, created_at
		 FROM bookings WHERE id = $1`,
		id,
	).Scan(&b.ID, &b.SlotID, &b.UserID, &b.Status, &b.ConferenceLink, &b.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &b, nil
}

func (r *BookingRepository) ListByUser(ctx context.Context, userID string) ([]domain.Booking, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT b.id, b.slot_id, b.user_id, b.status, b.conference_link, b.created_at
		 FROM bookings b
		 JOIN slots s ON s.id = b.slot_id
		 WHERE b.user_id = $1 AND s.start_at >= $2
		 ORDER BY s.start_at`,
		userID, time.Now().UTC(),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	bookings := make([]domain.Booking, 0)
	for rows.Next() {
		var b domain.Booking
		if err := rows.Scan(&b.ID, &b.SlotID, &b.UserID, &b.Status, &b.ConferenceLink, &b.CreatedAt); err != nil {
			return nil, err
		}
		bookings = append(bookings, b)
	}
	return bookings, rows.Err()
}

func (r *BookingRepository) Cancel(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE bookings SET status = 'cancelled' WHERE id = $1`,
		id,
	)
	return err
}
