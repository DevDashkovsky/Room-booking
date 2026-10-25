package handler

import (
	"context"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog/log"
)

// Keep database work within the server's 15-second write deadline.
const requestTimeout = 10 * time.Second

func withRequestDeadline(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
		defer cancel()
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func observeRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := chimiddleware.GetReqID(r.Context())
		if requestID != "" {
			w.Header().Set(chimiddleware.RequestIDHeader, requestID)
		}
		wrapped := chimiddleware.NewWrapResponseWriter(w, r.ProtoMajor)
		started := time.Now()

		defer func() {
			routePattern := ""
			if routeContext := chi.RouteContext(r.Context()); routeContext != nil {
				routePattern = routeContext.RoutePattern()
			}
			log.Info().
				Str("request_id", requestID).
				Str("method", r.Method).
				Str("path", r.URL.Path).
				Str("route", routePattern).
				Int("status", wrapped.Status()).
				Int("bytes", wrapped.BytesWritten()).
				Dur("duration", time.Since(started)).
				Msg("http request")
		}()

		defer func() {
			if recovered := recover(); recovered != nil {
				log.Error().
					Interface("panic", recovered).
					Bytes("stack", debug.Stack()).
					Str("request_id", requestID).
					Msg("panic recovered")
				if wrapped.Status() == 0 {
					respondError(wrapped, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
				}
			}
		}()

		next.ServeHTTP(wrapped, r)
	})
}
