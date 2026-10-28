package service_test

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/DevDashkovsky/room-booking/internal/domain"
	"github.com/DevDashkovsky/room-booking/internal/repository"
	"github.com/DevDashkovsky/room-booking/internal/service"
)

func moveBooking(t *testing.T, f *scheduleFixture, slot string) *domain.Booking {
	t.Helper()
	b, err := f.bookings.Create(t.Context(), slot, bookingUser, true)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func storedBooking(t *testing.T, f *scheduleFixture, id string) *domain.Booking {
	t.Helper()
	b, err := repository.NewBookingRepository(f.pool).GetByID(t.Context(), id)
	if err != nil || b == nil {
		t.Fatalf("read booking: %v %v", b, err)
	}
	return b
}

func TestPostgres_BookingMovePreservesIdentityAndReleasesSource(t *testing.T) {
	f := newScheduleFixture(t)
	slots := f.list(t)
	b := moveBooking(t, f, slots[0].ID)
	room := &domain.Room{Name: "move destination"}
	if err := repository.NewRoomRepository(f.pool).Create(t.Context(), room); err != nil {
		t.Fatal(err)
	}
	rule := f.replacement("09:00", "11:00")
	rule.RoomID = room.ID
	if err := f.schedules.Create(t.Context(), rule); err != nil {
		t.Fatal(err)
	}
	targets, err := f.slots.List(t.Context(), room.ID, f.date)
	if err != nil || len(targets) == 0 {
		t.Fatalf("target slots: %v %v", targets, err)
	}
	moved, err := f.bookings.Move(t.Context(), strings.ToUpper(b.ID), strings.ToUpper(targets[0].ID), bookingUser)
	if err != nil {
		t.Fatal(err)
	}
	want := *b
	want.SlotID = targets[0].ID
	if !reflect.DeepEqual(moved, &want) || !reflect.DeepEqual(storedBooking(t, f, b.ID), &want) {
		t.Fatalf("identity changed: before=%+v after=%+v", b, moved)
	}
	repeated, err := f.bookings.Move(t.Context(), b.ID, targets[0].ID, bookingUser)
	if err != nil || !reflect.DeepEqual(repeated, moved) {
		t.Fatalf("repeat: %+v %v", repeated, err)
	}
	moveBooking(t, f, slots[0].ID) // The old slot is bookable again.
}

func TestPostgres_BookingMoveFailuresPreserveBooking(t *testing.T) {
	for _, name := range []string{"occupied", "cancelled", "past source", "past target", "off grid", "foreign", "missing target", "missing booking", "same past slot"} {
		t.Run(name, func(t *testing.T) {
			f := newScheduleFixture(t)
			slots := f.list(t)
			b := moveBooking(t, f, slots[0].ID)
			id, target, owner := b.ID, slots[1].ID, bookingUser
			wantErr := domain.ErrBookingMoveConflict
			switch name {
			case "occupied":
				moveBooking(t, f, target)
				wantErr = domain.ErrSlotAlreadyBooked
			case "cancelled":
				if _, err := f.bookings.Cancel(t.Context(), b.ID, bookingUser); err != nil {
					t.Fatal(err)
				}
			case "past source", "past target", "same past slot":
				expired := slots[0].ID
				if name == "past target" {
					expired = target
				}
				if name == "same past slot" {
					target = expired
				}
				if _, err := f.pool.Exec(t.Context(), "UPDATE slots SET start_at=now()-interval '1 hour',end_at=now()-interval '30 minutes' WHERE id=$1", expired); err != nil {
					t.Fatal(err)
				}
			case "off grid":
				if err := f.schedules.Update(t.Context(), f.replacement("09:00", "09:30")); err != nil {
					t.Fatal(err)
				}
			case "foreign":
				owner = "00000000-0000-0000-0000-000000000001"
				wantErr = domain.ErrForbidden
			case "missing target":
				target = "00000000-0000-0000-0000-000000000004"
				wantErr = domain.ErrSlotNotFound
			case "missing booking":
				id = "00000000-0000-0000-0000-000000000004"
				wantErr = domain.ErrBookingNotFound
			}
			before := storedBooking(t, f, b.ID)
			if _, err := f.bookings.Move(t.Context(), id, target, owner); !errors.Is(err, wantErr) {
				t.Fatalf("got %v, want %v", err, wantErr)
			}
			if after := storedBooking(t, f, b.ID); !reflect.DeepEqual(before, after) {
				t.Fatalf("failed move changed booking: %+v -> %+v", before, after)
			}
		})
	}
}

func TestPostgres_BookingMoveCompetesForTarget(t *testing.T) {
	f := newScheduleFixture(t)
	slots := f.list(t)
	first, second := moveBooking(t, f, slots[0].ID), moveBooking(t, f, slots[1].ID)
	start := make(chan struct{})
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	for _, b := range []*domain.Booking{first, second} {
		wg.Go(func() {
			<-start
			_, err := f.bookings.Move(t.Context(), b.ID, slots[2].ID, bookingUser)
			errs <- err
		})
	}
	close(start)
	wg.Wait()
	close(errs)
	success := 0
	for err := range errs {
		if err == nil {
			success++
		} else if !errors.Is(err, domain.ErrSlotAlreadyBooked) {
			t.Fatal(err)
		}
	}
	if success != 1 {
		t.Fatalf("successful moves=%d", success)
	}
	for _, b := range []*domain.Booking{first, second} {
		actual := storedBooking(t, f, b.ID)
		if actual.SlotID != slots[2].ID && actual.SlotID != b.SlotID {
			t.Fatalf("loser lost original slot: %+v", actual)
		}
	}
}

func TestPostgres_BookingMoveSameBookingSerializes(t *testing.T) {
	f := newScheduleFixture(t)
	slots := f.list(t)
	b := moveBooking(t, f, slots[0].ID)
	start := make(chan struct{})
	errs := make(chan error, 8)
	var wg sync.WaitGroup
	for i := range 8 {
		wg.Go(func() {
			<-start
			_, err := f.bookings.Move(t.Context(), b.ID, slots[1+i%2].ID, bookingUser)
			errs <- err
		})
	}
	close(start)
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	after := storedBooking(t, f, b.ID)
	if after.SlotID != slots[1].ID && after.SlotID != slots[2].ID {
		t.Fatalf("wrong final slot: %+v", after)
	}
	var count int
	if err := f.pool.QueryRow(t.Context(), "SELECT count(*) FROM bookings WHERE status='active'").Scan(&count); err != nil || count != 1 {
		t.Fatalf("active count=%d: %v", count, err)
	}
}

func TestPostgres_BookingMoveOppositeOccupiedSlots(t *testing.T) {
	f := newScheduleFixture(t)
	slots := f.list(t)
	a, b := moveBooking(t, f, slots[0].ID), moveBooking(t, f, slots[1].ID)
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
	defer cancel()
	start := make(chan struct{})
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	wg.Go(func() { <-start; _, err := f.bookings.Move(ctx, a.ID, b.SlotID, bookingUser); errs <- err })
	wg.Go(func() { <-start; _, err := f.bookings.Move(ctx, b.ID, a.SlotID, bookingUser); errs <- err })
	close(start)
	wg.Wait()
	for range 2 {
		if err := <-errs; !errors.Is(err, domain.ErrSlotAlreadyBooked) {
			t.Fatalf("opposite moves must fail without deadlock: %v", err)
		}
	}
}

func TestPostgres_BookingMoveRacesCreate(t *testing.T) {
	f := newScheduleFixture(t)
	slots := f.list(t)
	for range 10 {
		b := moveBooking(t, f, slots[0].ID)
		var moved *domain.Booking
		var moveErr, createErr error
		start := make(chan struct{})
		var wg sync.WaitGroup
		wg.Go(func() { <-start; moved, moveErr = f.bookings.Move(t.Context(), b.ID, slots[1].ID, bookingUser) })
		wg.Go(func() { <-start; _, createErr = f.bookings.Create(t.Context(), slots[1].ID, bookingUser, false) })
		close(start)
		wg.Wait()
		switch {
		case moveErr == nil && errors.Is(createErr, domain.ErrSlotAlreadyBooked):
			if moved.SlotID != slots[1].ID {
				t.Fatal("wrong target")
			}
		case createErr == nil && errors.Is(moveErr, domain.ErrSlotAlreadyBooked):
			if storedBooking(t, f, b.ID).SlotID != slots[0].ID {
				t.Fatal("losing move lost source")
			}
		default:
			t.Fatalf("move=%v create=%v", moveErr, createErr)
		}
		if _, err := f.pool.Exec(t.Context(), "UPDATE bookings SET status='cancelled'"); err != nil {
			t.Fatal(err)
		}
	}
}

func TestPostgres_BookingMoveRacesSchedule(t *testing.T) {
	f := newScheduleFixture(t)
	slots := f.list(t)
	for range 10 {
		if err := f.schedules.Update(t.Context(), f.replacement("09:00", "11:00")); err != nil {
			t.Fatal(err)
		}
		b := moveBooking(t, f, slots[3].ID)
		var moveErr, updateErr error
		start := make(chan struct{})
		var wg sync.WaitGroup
		wg.Go(func() { <-start; _, moveErr = f.bookings.Move(t.Context(), b.ID, slots[0].ID, bookingUser) })
		wg.Go(func() { <-start; updateErr = f.schedules.Update(t.Context(), f.replacement("10:00", "11:00")) })
		close(start)
		wg.Wait()
		valid := (moveErr == nil && errors.Is(updateErr, domain.ErrScheduleBookingConflict)) ||
			(updateErr == nil && errors.Is(moveErr, domain.ErrBookingMoveConflict))
		if !valid {
			t.Fatalf("move=%v update=%v", moveErr, updateErr)
		}
		actual := storedBooking(t, f, b.ID)
		rule, err := f.schedules.Get(t.Context(), f.room.ID)
		if err != nil {
			t.Fatal(err)
		}
		slot, err := repository.NewSlotRepository(f.pool).GetByID(t.Context(), actual.SlotID)
		if err != nil || !rule.Allows(slot) {
			t.Fatalf("booking outside current schedule: %v", err)
		}
		if _, err := f.bookings.Cancel(t.Context(), b.ID, bookingUser); err != nil {
			t.Fatal(err)
		}
	}
}

func TestPostgres_BookingMoveRacesCancel(t *testing.T) {
	f := newScheduleFixture(t)
	slots := f.list(t)
	for range 10 {
		b := moveBooking(t, f, slots[0].ID)
		var cancelled *domain.Booking
		var moveErr, cancelErr error
		start := make(chan struct{})
		var wg sync.WaitGroup
		wg.Go(func() { <-start; _, moveErr = f.bookings.Move(t.Context(), b.ID, slots[1].ID, bookingUser) })
		wg.Go(func() { <-start; cancelled, cancelErr = f.bookings.Cancel(t.Context(), b.ID, bookingUser) })
		close(start)
		wg.Wait()
		if cancelErr != nil || (moveErr != nil && !errors.Is(moveErr, domain.ErrBookingMoveConflict)) {
			t.Fatalf("move=%v cancel=%v", moveErr, cancelErr)
		}
		actual := storedBooking(t, f, b.ID)
		if actual.Status != "cancelled" || !reflect.DeepEqual(actual, cancelled) {
			t.Fatalf("cancel response stale: got=%+v stored=%+v", cancelled, actual)
		}
	}
}

type cancelReadBarrier struct {
	*repository.BookingRepository
	read, resume chan struct{}
}

func (r *cancelReadBarrier) GetByID(ctx context.Context, id string) (*domain.Booking, error) {
	b, err := r.BookingRepository.GetByID(ctx, id)
	close(r.read)
	select {
	case <-r.resume:
		return b, err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func TestPostgres_CancelReturnsMoveCommittedAfterInitialRead(t *testing.T) {
	f := newScheduleFixture(t)
	slots := f.list(t)
	b := moveBooking(t, f, slots[0].ID)
	barrier := &cancelReadBarrier{BookingRepository: repository.NewBookingRepository(f.pool), read: make(chan struct{}), resume: make(chan struct{})}
	svc := service.NewBookingService(barrier, nil)
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	var result *domain.Booking
	var cancelErr error
	var wg sync.WaitGroup
	wg.Go(func() { result, cancelErr = svc.Cancel(ctx, b.ID, bookingUser) })
	select {
	case <-barrier.read:
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	_, moveErr := f.bookings.Move(ctx, b.ID, slots[1].ID, bookingUser)
	close(barrier.resume)
	wg.Wait()
	if moveErr != nil || cancelErr != nil || result.SlotID != slots[1].ID || result.Status != "cancelled" {
		t.Fatalf("move=%v cancel=%v result=%+v", moveErr, cancelErr, result)
	}
}

func TestPostgres_BookingMoveRoomLockDeadlineRollsBack(t *testing.T) {
	f := newScheduleFixture(t)
	slots := f.list(t)
	b := moveBooking(t, f, slots[0].ID)
	tx, err := f.pool.Begin(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(t.Context()) }()
	if _, err := tx.Exec(t.Context(), "SELECT id FROM rooms WHERE id=$1 FOR UPDATE", f.room.ID); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
	defer cancel()
	if _, err := f.bookings.Move(ctx, b.ID, slots[1].ID, bookingUser); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("blocked move=%v", err)
	}
	if err := tx.Rollback(t.Context()); err != nil {
		t.Fatal(err)
	}
	if storedBooking(t, f, b.ID).SlotID != slots[0].ID {
		t.Fatal("timed out move changed booking")
	}
	if _, err := f.bookings.Move(t.Context(), b.ID, slots[1].ID, bookingUser); err != nil {
		t.Fatalf("retry after rollback: %v", err)
	}
}

func TestPostgres_BookingMoveWaitsForUncommittedTarget(t *testing.T) {
	for _, commit := range []bool{true, false} {
		name := "deadline"
		if commit {
			name = "conflict after commit"
		}
		t.Run(name, func(t *testing.T) {
			f := newScheduleFixture(t)
			slots := f.list(t)
			b := moveBooking(t, f, slots[0].ID)
			tx, err := f.pool.Begin(t.Context())
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = tx.Rollback(t.Context()) }()
			var blockerPID int
			if err := tx.QueryRow(t.Context(), "SELECT pg_backend_pid()").Scan(&blockerPID); err != nil {
				t.Fatal(err)
			}
			// This row is invisible to Move's EXISTS, but the unique index must
			// wait for this transaction before allowing a competing UPDATE.
			if _, err := tx.Exec(t.Context(), "INSERT INTO bookings(slot_id,user_id) VALUES($1,$2)", slots[1].ID, bookingUser); err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
			defer cancel()
			result := make(chan error, 1)
			go func() {
				_, err := f.bookings.Move(ctx, b.ID, slots[1].ID, bookingUser)
				result <- err
			}()
			waitCtx, stopWaiting := context.WithTimeout(t.Context(), 2*time.Second)
			defer stopWaiting()
			ticker := time.NewTicker(10 * time.Millisecond)
			defer ticker.Stop()
			for {
				var blocked bool
				err := f.pool.QueryRow(waitCtx, `SELECT EXISTS(
 SELECT 1 FROM pg_stat_activity WHERE $1=ANY(pg_blocking_pids(pid))
 AND query LIKE 'UPDATE bookings SET slot_id=%')`, blockerPID).Scan(&blocked)
				if err != nil {
					t.Fatalf("observe unique-index wait: %v", err)
				}
				if blocked {
					break
				}
				select {
				case <-ticker.C:
				case <-waitCtx.Done():
					t.Fatal("move never waited for the uncommitted target")
				}
			}
			wantErr := context.DeadlineExceeded
			if commit {
				if err := tx.Commit(t.Context()); err != nil {
					t.Fatal(err)
				}
				wantErr = domain.ErrSlotAlreadyBooked
			}
			select {
			case err := <-result:
				if !errors.Is(err, wantErr) {
					t.Fatalf("move error=%v, want %v", err, wantErr)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("move did not finish after its deadline")
			}
			if actual := storedBooking(t, f, b.ID); !reflect.DeepEqual(actual, b) {
				t.Fatalf("unique-index wait changed original booking: %+v", actual)
			}
		})
	}
}
