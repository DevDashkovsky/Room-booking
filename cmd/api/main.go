package main

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/rs/zerolog"

	"github.com/DevDashkovsky/room-booking/internal/config"
	"github.com/DevDashkovsky/room-booking/internal/db"
	"github.com/DevDashkovsky/room-booking/internal/handler"
	"github.com/DevDashkovsky/room-booking/internal/repository"
	"github.com/DevDashkovsky/room-booking/internal/service"
)

func main() {
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()

	cfg := config.Load()
	ctx := context.Background()

	var pool *db.Pool
	for i := 0; i < 30; i++ {
		p, err := db.Connect(ctx, cfg.DatabaseURL)
		if err == nil {
			pool = p
			break
		}
		logger.Warn().Err(err).Msgf("waiting for db... (%d/30)", i+1)
		time.Sleep(time.Second)
	}
	if pool == nil {
		logger.Fatal().Msg("failed to connect to database")
	}
	defer pool.Close()

	if err := db.RunMigrations(cfg.DatabaseURL, "migrations"); err != nil {
		logger.Fatal().Err(err).Msg("failed to run migrations")
	}

	roomRepo := repository.NewRoomRepository(pool.PgxPool())
	scheduleRepo := repository.NewScheduleRepository(pool.PgxPool())
	slotRepo := repository.NewSlotRepository(pool.PgxPool())
	bookingRepo := repository.NewBookingRepository(pool.PgxPool())

	roomSvc := service.NewRoomService(roomRepo)
	scheduleSvc := service.NewScheduleService(scheduleRepo, roomRepo, slotRepo)
	slotSvc := service.NewSlotService(slotRepo, roomRepo, scheduleRepo)
	bookingSvc := service.NewBookingService(bookingRepo, slotRepo)

	r := handler.NewRouter(handler.Deps{
		JWTSecret: cfg.JWTSecret,
		AuthH:     handler.NewAuthHandler(cfg.JWTSecret),
		RoomH:     handler.NewRoomHandler(roomSvc),
		ScheduleH: handler.NewScheduleHandler(scheduleSvc),
		SlotH:     handler.NewSlotHandler(slotSvc),
		BookingH:  handler.NewBookingHandler(bookingSvc),
	})

	logger.Info().Str("port", cfg.Port).Msg("starting server")
	if err := http.ListenAndServe(":"+cfg.Port, r); err != nil {
		logger.Fatal().Err(err).Msg("server failed")
	}
}
