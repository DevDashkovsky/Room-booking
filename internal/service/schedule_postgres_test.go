package service_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/DevDashkovsky/room-booking/internal/domain"
	"github.com/DevDashkovsky/room-booking/internal/repository"
	"github.com/DevDashkovsky/room-booking/internal/service"
	"github.com/jackc/pgx/v5/pgxpool"
)

type scheduleFixture struct {
	pool      *pgxpool.Pool
	room      *domain.Room
	schedule  *domain.Schedule
	schedules *service.ScheduleService
	slots     *service.SlotService
	bookings  *service.BookingService
	date      time.Time
}

func newScheduleFixture(t *testing.T) *scheduleFixture {
	t.Helper()
	pool := testPool(t)
	rooms := repository.NewRoomRepository(pool)
	schedules := repository.NewScheduleRepository(pool)
	slots := repository.NewSlotRepository(pool)
	f := &scheduleFixture{pool: pool, room: &domain.Room{Name: "editable"}, schedules: service.NewScheduleService(schedules, rooms), slots: service.NewSlotService(slots, rooms), bookings: service.NewBookingService(repository.NewBookingRepository(pool), slots), date: time.Now().UTC().AddDate(0, 0, 2)}
	if err := rooms.Create(t.Context(), f.room); err != nil {
		t.Fatal(err)
	}
	f.schedule = f.replacement("09:00", "11:00")
	if err := f.schedules.Create(t.Context(), f.schedule); err != nil {
		t.Fatal(err)
	}
	return f
}
func (f *scheduleFixture) replacement(start, end string) *domain.Schedule {
	return &domain.Schedule{RoomID: f.room.ID, DaysOfWeek: []int{1, 2, 3, 4, 5, 6, 7}, StartTime: start, EndTime: end}
}
func (f *scheduleFixture) list(t *testing.T) []domain.Slot {
	t.Helper()
	slots, err := f.slots.List(t.Context(), f.room.ID, f.date)
	if err != nil {
		t.Fatal(err)
	}
	return slots
}

const bookingUser = "00000000-0000-0000-0000-000000000002"

func TestPostgres_ScheduleUpdatePreservesBookingsAndSlotIDs(t *testing.T) {
	f := newScheduleFixture(t)
	original := f.list(t)
	booking, err := f.bookings.Create(t.Context(), original[0].ID, bookingUser, false)
	if err != nil {
		t.Fatal(err)
	}
	for _, candidate := range []*domain.Schedule{f.replacement("10:00", "11:00"), f.replacement("08:15", "11:15"), {RoomID: f.room.ID, DaysOfWeek: []int{int(f.date.Weekday())%7 + 1}, StartTime: "09:00", EndTime: "11:00"}} {
		if err := f.schedules.Update(t.Context(), candidate); !errors.Is(err, domain.ErrScheduleBookingConflict) {
			t.Fatalf("conflicting update: %v", err)
		}
		saved, err := repository.NewScheduleRepository(f.pool).GetByRoomID(t.Context(), f.room.ID)
		if err != nil || saved.StartTime != "09:00" || saved.EndTime != "11:00" {
			t.Fatalf("conflict changed schedule: %+v %v", saved, err)
		}
	}
	compatible := f.replacement("08:00", "11:00")
	compatible.RoomID = strings.ToUpper(f.room.ID)
	if err := f.schedules.Update(t.Context(), compatible); err != nil {
		t.Fatal(err)
	}
	if compatible.ID != f.schedule.ID {
		t.Fatal("schedule ID changed")
	}
	if _, err := f.bookings.Cancel(t.Context(), booking.ID, bookingUser); err != nil {
		t.Fatal(err)
	}
	available := f.list(t)
	if len(available) != 6 || available[2].ID != original[0].ID {
		t.Fatalf("compatible slots changed: %+v", available)
	}
	if err := f.schedules.Update(t.Context(), f.replacement("10:00", "11:00")); err != nil {
		t.Fatal(err)
	}
	available = f.list(t)
	if len(available) != 2 || available[0].ID != original[2].ID {
		t.Fatalf("new availability: %+v", available)
	}
	if _, err := f.bookings.Create(t.Context(), original[0].ID, bookingUser, false); !errors.Is(err, domain.ErrInvalidRequest) {
		t.Fatalf("obsolete slot accepted: %v", err)
	}
	history, err := f.bookings.ListMy(t.Context(), bookingUser)
	if err != nil || len(history) != 1 || history[0].ID != booking.ID || history[0].Status != "cancelled" {
		t.Fatalf("history lost: %+v %v", history, err)
	}
	if err := f.schedules.Update(t.Context(), f.replacement("10:00", "11:00")); err != nil {
		t.Fatal(err)
	}
	again := f.list(t)
	if len(again) != 2 || again[0].ID != available[0].ID {
		t.Fatal("repeated update changed slots")
	}
}

func TestPostgres_ScheduleUpdateMissingAndOngoing(t *testing.T) {
	f := newScheduleFixture(t)
	missing := f.replacement("09:00", "11:00")
	missing.RoomID = "00000000-0000-0000-0000-000000000099"
	if err := f.schedules.Update(t.Context(), missing); !errors.Is(err, domain.ErrRoomNotFound) {
		t.Fatalf("missing room: %v", err)
	}
	room := &domain.Room{Name: "no schedule"}
	if err := repository.NewRoomRepository(f.pool).Create(t.Context(), room); err != nil {
		t.Fatal(err)
	}
	missing.RoomID = room.ID
	if err := f.schedules.Update(t.Context(), missing); !errors.Is(err, domain.ErrScheduleNotFound) {
		t.Fatalf("missing schedule: %v", err)
	}
	start := time.Now().UTC().Truncate(time.Minute).Add(-time.Minute)
	var id string
	if err := f.pool.QueryRow(t.Context(), "INSERT INTO slots(room_id,start_at,end_at) VALUES($1,$2,$3) RETURNING id", f.room.ID, start, start.Add(30*time.Minute)).Scan(&id); err != nil {
		t.Fatal(err)
	}
	if _, err := f.pool.Exec(t.Context(), "INSERT INTO bookings(slot_id,user_id) VALUES($1,$2)", id, bookingUser); err != nil {
		t.Fatal(err)
	}
	// A one-minute window contains no full slot, and must conflict even after a booking starts.
	if err := f.schedules.Update(t.Context(), f.replacement("00:00", "00:01")); !errors.Is(err, domain.ErrScheduleBookingConflict) {
		t.Fatalf("ongoing booking unprotected: %v", err)
	}
	if _, err := f.pool.Exec(t.Context(), "UPDATE slots SET start_at=start_at-interval '1 day',end_at=end_at-interval '1 day' WHERE id=$1", id); err != nil {
		t.Fatal(err)
	}
	if err := f.schedules.Update(t.Context(), f.replacement("00:00", "00:01")); err != nil {
		t.Fatalf("past booking blocks update: %v", err)
	}
}

func TestPostgres_ScheduleUpdateRacesBooking(t *testing.T) {
	f := newScheduleFixture(t)
	original := f.list(t)
	for range 12 {
		if err := f.schedules.Update(t.Context(), f.replacement("09:00", "11:00")); err != nil {
			t.Fatal(err)
		}
		start := make(chan struct{})
		var wg sync.WaitGroup
		var booking *domain.Booking
		var bookingErr, updateErr error
		wg.Go(func() {
			<-start
			booking, bookingErr = f.bookings.Create(t.Context(), original[0].ID, bookingUser, false)
		})
		wg.Go(func() { <-start; updateErr = f.schedules.Update(t.Context(), f.replacement("10:00", "11:00")) })
		close(start)
		wg.Wait()
		switch {
		case bookingErr == nil && errors.Is(updateErr, domain.ErrScheduleBookingConflict):
			if _, err := f.bookings.Cancel(t.Context(), booking.ID, bookingUser); err != nil {
				t.Fatal(err)
			}
		case updateErr == nil && errors.Is(bookingErr, domain.ErrInvalidRequest):
		default:
			t.Fatalf("inconsistent race: booking=%v update=%v", bookingErr, updateErr)
		}
	}
}

func TestPostgres_ScheduleUpdateRacesSlotGeneration(t *testing.T) {
	f := newScheduleFixture(t)
	start := make(chan struct{})
	var wg sync.WaitGroup
	errs := make(chan error, 13)
	results := make(chan []domain.Slot, 12)
	wg.Go(func() { <-start; errs <- f.schedules.Update(t.Context(), f.replacement("09:15", "11:15")) })
	for range 12 {
		wg.Go(func() {
			<-start
			slots, err := f.slots.List(t.Context(), f.room.ID, f.date)
			errs <- err
			results <- slots
		})
	}
	close(start)
	wg.Wait()
	close(errs)
	close(results)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	old, newSchedule := f.replacement("09:00", "11:00"), f.replacement("09:15", "11:15")
	for slots := range results {
		if len(slots) != 4 {
			t.Fatalf("incomplete list: %+v", slots)
		}
		expected := old
		if newSchedule.Allows(&slots[0]) {
			expected = newSchedule
		}
		for _, slot := range slots {
			if !expected.Allows(&slot) {
				t.Fatalf("mixed grids: %+v", slots)
			}
		}
	}
	if slots := f.list(t); len(slots) != 4 || !newSchedule.Allows(&slots[0]) {
		t.Fatalf("stale final list: %+v", slots)
	}
}

func TestPostgres_ScheduleUpdateWaitsForBookingLock(t *testing.T) {
	f := newScheduleFixture(t)
	slot := f.list(t)[0]
	tx, err := f.pool.Begin(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	if _, err := tx.Exec(t.Context(), "SELECT id FROM rooms WHERE id=$1 FOR SHARE", f.room.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(t.Context(), "INSERT INTO bookings(slot_id,user_id) VALUES($1,$2)", slot.ID, bookingUser); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
	defer cancel()
	if err := f.schedules.Update(ctx, f.replacement("10:00", "11:00")); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("update bypassed booking lock: %v", err)
	}
	if err := tx.Commit(t.Context()); err != nil {
		t.Fatal(err)
	}
	if err := f.schedules.Update(t.Context(), f.replacement("10:00", "11:00")); !errors.Is(err, domain.ErrScheduleBookingConflict) {
		t.Fatalf("committed booking missed: %v", err)
	}
}

func TestPostgres_BookingAndListWaitForScheduleUpdate(t *testing.T) {
	f := newScheduleFixture(t)
	original := f.list(t)
	tx, err := f.pool.Begin(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	if _, err := tx.Exec(t.Context(), "SELECT id FROM rooms WHERE id=$1 FOR UPDATE", f.room.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(t.Context(), "UPDATE schedules SET start_time='10:00' WHERE room_id=$1", f.room.ID); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
	_, err = f.bookings.Create(ctx, original[0].ID, bookingUser, false)
	cancel()
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("booking bypassed update lock: %v", err)
	}
	ctx, cancel = context.WithTimeout(t.Context(), 100*time.Millisecond)
	_, err = f.slots.List(ctx, f.room.ID, f.date)
	cancel()
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("list bypassed update lock: %v", err)
	}
	if err := tx.Commit(t.Context()); err != nil {
		t.Fatal(err)
	}
	if _, err := f.bookings.Create(t.Context(), original[0].ID, bookingUser, false); !errors.Is(err, domain.ErrInvalidRequest) {
		t.Fatalf("obsolete slot accepted: %v", err)
	}
	if available := f.list(t); len(available) != 2 || available[0].ID != original[2].ID {
		t.Fatalf("incomplete current list: %+v", available)
	}
}
