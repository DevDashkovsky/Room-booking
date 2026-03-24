package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/DevDashkovsky/room-booking/internal/domain"
)

type RoomRepository struct {
	pool *pgxpool.Pool
}

func NewRoomRepository(pool *pgxpool.Pool) *RoomRepository {
	return &RoomRepository{pool: pool}
}

func (r *RoomRepository) Create(ctx context.Context, room *domain.Room) error {
	return r.pool.QueryRow(ctx,
		`INSERT INTO rooms (name, description, capacity)
		 VALUES ($1, $2, $3)
		 RETURNING id, created_at`,
		room.Name, room.Description, room.Capacity,
	).Scan(&room.ID, &room.CreatedAt)
}

func (r *RoomRepository) List(ctx context.Context) ([]domain.Room, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, name, description, capacity, created_at
		 FROM rooms
		 ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rooms []domain.Room
	for rows.Next() {
		var room domain.Room
		if err := rows.Scan(&room.ID, &room.Name, &room.Description, &room.Capacity, &room.CreatedAt); err != nil {
			return nil, err
		}
		rooms = append(rooms, room)
	}

	return rooms, rows.Err()
}

func (r *RoomRepository) GetByID(ctx context.Context, id string) (*domain.Room, error) {
	var room domain.Room
	err := r.pool.QueryRow(ctx,
		`SELECT id, name, description, capacity, created_at
		 FROM rooms WHERE id = $1`,
		id,
	).Scan(&room.ID, &room.Name, &room.Description, &room.Capacity, &room.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &room, nil
}
