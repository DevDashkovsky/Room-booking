package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/DevDashkovsky/room-booking/internal/config"
	"github.com/DevDashkovsky/room-booking/internal/db"
	"github.com/DevDashkovsky/room-booking/internal/handler"
	"github.com/DevDashkovsky/room-booking/internal/repository"
	"github.com/DevDashkovsky/room-booking/internal/service"
)

func main() {
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	// Делаем тот же логгер глобальным, чтобы handler-слой (handleServiceError)
	// писал внутренние ошибки в общий поток.
	log.Logger = logger

	cfg := config.Load()
	ctx := context.Background()

	var pool *db.Pool
	for i := 0; i < 30; i++ {
		p, err := db.Connect(ctx, cfg.DatabaseURL, cfg.DBMaxConns)
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

	userRepo := repository.NewUserRepository(pool.PgxPool())
	roomRepo := repository.NewRoomRepository(pool.PgxPool())
	scheduleRepo := repository.NewScheduleRepository(pool.PgxPool())
	slotRepo := repository.NewSlotRepository(pool.PgxPool())
	bookingRepo := repository.NewBookingRepository(pool.PgxPool())

	authSvc := service.NewAuthService(userRepo, cfg.JWTSecret)
	roomSvc := service.NewRoomService(roomRepo)
	scheduleSvc := service.NewScheduleService(scheduleRepo, roomRepo, slotRepo)
	slotSvc := service.NewSlotService(slotRepo, roomRepo, scheduleRepo)
	bookingSvc := service.NewBookingService(bookingRepo, slotRepo)

	r := handler.NewRouter(handler.Deps{
		JWTSecret: cfg.JWTSecret,
		AuthH:     handler.NewAuthHandler(cfg.JWTSecret, authSvc),
		RoomH:     handler.NewRoomHandler(roomSvc),
		ScheduleH: handler.NewScheduleHandler(scheduleSvc),
		SlotH:     handler.NewSlotHandler(slotSvc),
		BookingH:  handler.NewBookingHandler(bookingSvc),
		Ping:      pool.PgxPool().Ping,
	})

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	shutdownCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		logger.Info().Str("port", cfg.Port).Msg("starting server")
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Fatal().Err(err).Msg("server failed")
		}
	}()

	<-shutdownCtx.Done()
	logger.Info().Msg("shutdown signal received, draining connections")

	shutdownTimeout, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownTimeout); err != nil {
		logger.Error().Err(err).Msg("graceful shutdown failed")
	}
}
