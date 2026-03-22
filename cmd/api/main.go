package main

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/DevDashkovsky/room-booking/internal/config"
	"github.com/DevDashkovsky/room-booking/internal/db"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"
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

	r := chi.NewRouter()

	r.Get("/_info", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	logger.Info().Str("port", cfg.Port).Msg("starting server")
	if err := http.ListenAndServe(":"+cfg.Port, r); err != nil {
		logger.Fatal().Err(err).Msg("server failed")
	}
}
