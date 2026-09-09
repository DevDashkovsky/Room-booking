package service_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/DevDashkovsky/room-booking/internal/db"
	"github.com/DevDashkovsky/room-booking/internal/domain"
	"github.com/DevDashkovsky/room-booking/internal/repository"
	"github.com/DevDashkovsky/room-booking/internal/service"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("ROOM_BOOKING_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("ROOM_BOOKING_TEST_DATABASE_URL is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	admin, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(admin.Close)
	var randomID string
	if err := admin.QueryRow(ctx, "SELECT gen_random_uuid()::text").Scan(&randomID); err != nil {
		t.Fatal(err)
	}
	schema := "room_booking_test_" + strings.ReplaceAll(randomID, "-", "")
	identifier := pgx.Identifier{schema}.Sanitize()
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+identifier); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if _, err := admin.Exec(ctx, "DROP SCHEMA "+identifier+" CASCADE"); err != nil {
			t.Errorf("cleanup schema: %v", err)
		}
	})
	databaseURL, err := url.Parse(dsn)
	if err != nil {
		t.Fatal(err)
	}
	query := databaseURL.Query()
	query.Set("search_path", schema)
	databaseURL.RawQuery = query.Encode()
	connection, err := db.Connect(ctx, databaseURL.String(), 20)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(connection.Close)
	pool := connection.PgxPool()
	migrations, err := filepath.Glob("../../migrations/*.sql")
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range migrations {
		content, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		up := strings.Split(string(content), "-- +goose Down")[0]
		if _, err := pool.Exec(ctx, up); err != nil {
			t.Fatalf("migration %s: %v", file, err)
		}
	}
	return pool
}

func TestPostgres_LazySlotsAndConcurrentBooking(t *testing.T) {
	pool := testPool(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	rooms := repository.NewRoomRepository(pool)
	schedules := repository.NewScheduleRepository(pool)
	slots := repository.NewSlotRepository(pool)
	bookings := repository.NewBookingRepository(pool)
	scheduleSvc := service.NewScheduleService(schedules, rooms)
	slotSvc := service.NewSlotService(slots, rooms, schedules)
	bookingSvc := service.NewBookingService(bookings, slots)
	room := &domain.Room{Name: "concurrent room"}
	if err := rooms.Create(ctx, room); err != nil {
		t.Fatal(err)
	}
	date := time.Now().UTC().AddDate(0, 0, 90)
	if got, err := slotSvc.List(ctx, room.ID, date); err != nil || len(got) != 0 {
		t.Fatalf("without schedule: %v %v", got, err)
	}
	const requests = 12
	errs := make(chan error, requests)
	var wg sync.WaitGroup
	for range requests {
		wg.Go(func() {
			errs <- scheduleSvc.Create(ctx, &domain.Schedule{RoomID: room.ID, DaysOfWeek: []int{1, 2, 3, 4, 5, 6, 7}, StartTime: "9:00", EndTime: "10:15"})
		})
	}
	wg.Wait()
	close(errs)
	created := 0
	for err := range errs {
		if err == nil {
			created++
		} else if !errors.Is(err, domain.ErrScheduleExists) {
			t.Errorf("concurrent schedule: %v", err)
		}
	}
	if created != 1 {
		t.Fatalf("created schedules = %d", created)
	}
	var count int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM slots").Scan(&count); err != nil || count != 0 {
		t.Fatalf("eager slots: %d %v", count, err)
	}
	schedule, err := schedules.GetByRoomID(ctx, room.ID)
	if err != nil {
		t.Fatal(err)
	}
	if schedule.StartTime != "09:00" || schedule.EndTime != "10:15" {
		t.Fatalf("TIME round trip: %#v", schedule)
	}
	results := make(chan []domain.Slot, requests)
	errs = make(chan error, requests)
	for range requests {
		wg.Go(func() { got, err := slotSvc.List(ctx, room.ID, date); results <- got; errs <- err })
	}
	wg.Wait()
	close(results)
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	var first []domain.Slot
	for got := range results {
		if len(got) != 2 {
			t.Fatalf("slots = %d, want 2 full slots", len(got))
		}
		if first == nil {
			first = got
		}
		for i := range got {
			if got[i].ID != first[i].ID {
				t.Fatal("slot IDs changed")
			}
		}
	}
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM slots").Scan(&count); err != nil || count != 2 {
		t.Fatalf("unexpected generation scope: %d %v", count, err)
	}
	errs = make(chan error, requests)
	winners := make(chan *domain.Booking, requests)
	const userID = "00000000-0000-0000-0000-000000000002"
	for range requests {
		wg.Go(func() {
			booking, err := bookingSvc.Create(ctx, first[0].ID, userID, false)
			errs <- err
			if err == nil {
				winners <- booking
			}
		})
	}
	wg.Wait()
	close(errs)
	close(winners)
	for err := range errs {
		if err != nil && !errors.Is(err, domain.ErrSlotAlreadyBooked) {
			t.Errorf("concurrent booking: %v", err)
		}
	}
	if len(winners) != 1 {
		t.Fatalf("booking winners=%d", len(winners))
	}
	booking := <-winners
	available, err := slotSvc.List(ctx, room.ID, date)
	if err != nil || len(available) != 1 {
		t.Fatalf("booked slot still available: %v %v", available, err)
	}
	if _, err := bookingSvc.Cancel(ctx, booking.ID, "00000000-0000-0000-0000-000000000001"); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("cancel by nonowner: %v", err)
	}
	for range 2 {
		got, err := bookingSvc.Cancel(ctx, booking.ID, userID)
		if err != nil || got.Status != "cancelled" {
			t.Fatalf("idempotent cancel: %v %v", got, err)
		}
	}
	my, err := bookingSvc.ListMy(ctx, userID)
	if err != nil || len(my) != 1 || my[0].Status != "cancelled" {
		t.Fatalf("cancelled future booking missing: %v %v", my, err)
	}
	available, err = slotSvc.List(ctx, room.ID, date)
	if err != nil || len(available) != 2 || available[0].ID != first[0].ID {
		t.Fatalf("cancel did not release stable slot: %v %v", available, err)
	}
	if _, err := bookingSvc.Create(ctx, first[0].ID, userID, false); err != nil {
		t.Fatalf("rebook: %v", err)
	}
	if _, err := pool.Exec(ctx, "UPDATE slots SET start_at=start_at-interval '180 days', end_at=end_at-interval '180 days'"); err != nil {
		t.Fatal(err)
	}
	if _, err := bookingSvc.Create(ctx, first[1].ID, userID, false); !errors.Is(err, domain.ErrInvalidRequest) {
		t.Fatalf("past booking: %v", err)
	}
	my, err = bookingSvc.ListMy(ctx, userID)
	if err != nil || len(my) != 0 {
		t.Fatalf("past bookings returned: %v %v", my, err)
	}
	past, err := slotSvc.List(ctx, room.ID, time.Now().UTC().AddDate(0, 0, -90))
	if err != nil || len(past) != 0 {
		t.Fatalf("past slots available: %v %v", past, err)
	}
}

func TestPostgres_AuthAndStablePagination(t *testing.T) {
	pool := testPool(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	auth := service.NewAuthService(repository.NewUserRepository(pool), "test-secret")
	for _, role := range []string{"admin", "user"} {
		email := role + "@example.com"
		user, err := auth.Register(ctx, email, "password", role)
		if err != nil || user.Role != role {
			t.Fatalf("register %s: %v %v", role, user, err)
		}
		if _, err := auth.Login(ctx, email, "password"); err != nil {
			t.Fatal(err)
		}
		if _, err := auth.Register(ctx, email, "password", role); !errors.Is(err, domain.ErrEmailExists) {
			t.Fatalf("duplicate email: %v", err)
		}
		if _, err := auth.Login(ctx, email, "wrong"); !errors.Is(err, domain.ErrUnauthorized) {
			t.Fatalf("wrong password: %v", err)
		}
	}
	var roomID string
	if err := pool.QueryRow(ctx, "INSERT INTO rooms(name) VALUES ('pagination') RETURNING id").Scan(&roomID); err != nil {
		t.Fatal(err)
	}
	for i := 1; i <= 3; i++ {
		var slotID string
		if err := pool.QueryRow(ctx, "INSERT INTO slots(room_id,start_at,end_at) VALUES ($1,NOW()+$2*interval '1 hour',NOW()+($2+1)*interval '1 hour') RETURNING id", roomID, i).Scan(&slotID); err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx, "INSERT INTO bookings(id,slot_id,user_id,created_at) VALUES ($1,$2,'00000000-0000-0000-0000-000000000002','2026-01-01')", fmt.Sprintf("00000000-0000-0000-0000-%012d", i), slotID); err != nil {
			t.Fatal(err)
		}
	}
	svc := service.NewBookingService(repository.NewBookingRepository(pool), repository.NewSlotRepository(pool))
	for page := 1; page <= 3; page++ {
		result, err := svc.ListAll(ctx, page, 1)
		if err != nil {
			t.Fatal(err)
		}
		want := fmt.Sprintf("00000000-0000-0000-0000-%012d", 4-page)
		if len(result.Bookings) != 1 || result.Bookings[0].ID != want || result.Pagination.Total != 3 {
			t.Fatalf("page %d: %#v", page, result)
		}
	}
}

func TestPostgres_TimestampsRemainUTC(t *testing.T) {
	original := time.Local
	time.Local = time.FixedZone("UTC+7", 7*60*60)
	t.Cleanup(func() { time.Local = original })
	pool := testPool(t)
	var value time.Time
	if err := pool.QueryRow(t.Context(), "SELECT '2026-09-09 12:00:00+00'::timestamptz").Scan(&value); err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if string(encoded) != `"2026-09-09T12:00:00Z"` {
		t.Fatalf("non-UTC wire timestamp: %s", encoded)
	}
}
