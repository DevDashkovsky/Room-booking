package repository

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/DevDashkovsky/room-booking/internal/domain"
)

func (r *BookingRepository) Move(ctx context.Context, bookingID, slotID, userID string) (*domain.Booking, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Lock the booking before reading its source slot: another move may have
	// changed it while we waited. Cancel and other moves serialize on this row.
	var booking domain.Booking
	var owned bool
	err = tx.QueryRow(ctx, `SELECT id,slot_id,user_id,status,conference_link,created_at,user_id=$2
 FROM bookings WHERE id=$1 FOR UPDATE`, bookingID, userID).
		Scan(&booking.ID, &booking.SlotID, &booking.UserID, &booking.Status, &booking.ConferenceLink, &booking.CreatedAt, &owned)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrBookingNotFound
	}
	if err != nil {
		return nil, err
	}
	if !owned {
		return nil, domain.ErrForbidden
	}
	if booking.Status != "active" {
		return nil, domain.ErrBookingMoveConflict
	}
	var sourceStart time.Time
	if err := tx.QueryRow(ctx, "SELECT start_at FROM slots WHERE id=$1", booking.SlotID).Scan(&sourceStart); err != nil {
		return nil, err
	}
	var target domain.Slot
	err = tx.QueryRow(ctx, "SELECT id,room_id,start_at,end_at FROM slots WHERE id=$1", slotID).
		Scan(&target.ID, &target.RoomID, &target.Start, &target.End)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrSlotNotFound
	}
	if err != nil {
		return nil, err
	}
	// Schedule.Update locks the room, then reads bookings WITHOUT row locks.
	// Keep that order free of a room -> booking lock dependency.
	if _, err := lockRoom(ctx, tx, target.RoomID, false); err != nil {
		return nil, err
	}
	schedule, err := getSchedule(ctx, tx, target.RoomID)
	if err != nil {
		return nil, err
	}
	var now time.Time
	if err := tx.QueryRow(ctx, "SELECT clock_timestamp()").Scan(&now); err != nil {
		return nil, err
	}
	if !sourceStart.After(now) || !target.Start.After(now) || schedule == nil || !schedule.Allows(&target) {
		return nil, domain.ErrBookingMoveConflict
	}
	if booking.SlotID == target.ID {
		return &booking, nil
	}

	// Do not lock another booking here: opposite moves between occupied slots
	// must fail immediately instead of waiting on each other's unique keys.
	var occupied bool
	if err := tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM bookings WHERE slot_id=$1 AND status='active')", target.ID).Scan(&occupied); err != nil {
		return nil, err
	}
	if occupied {
		return nil, domain.ErrSlotAlreadyBooked
	}
	err = tx.QueryRow(ctx, `UPDATE bookings SET slot_id=$2
 WHERE id=$1 AND status='active' AND $3::timestamptz>clock_timestamp() AND $4::timestamptz>clock_timestamp()
 RETURNING id,slot_id,user_id,status,conference_link,created_at`, booking.ID, target.ID, sourceStart, target.Start).
		Scan(&booking.ID, &booking.SlotID, &booking.UserID, &booking.Status, &booking.ConferenceLink, &booking.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrBookingMoveConflict
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation && pgErr.ConstraintName == "uniq_active_booking_per_slot" {
		return nil, domain.ErrSlotAlreadyBooked
	}
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &booking, nil
}
