package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/DevDashkovsky/room-booking/internal/middleware"
)

type Deps struct {
	JWTSecret string
	AuthH     *AuthHandler
	RoomH     *RoomHandler
	ScheduleH *ScheduleHandler
	SlotH     *SlotHandler
	BookingH  *BookingHandler
	Ping      func(context.Context) error
}

func NewRouter(d Deps) http.Handler {
	r := chi.NewRouter()

	r.Get("/_info", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	r.Get("/ready", func(w http.ResponseWriter, req *http.Request) {
		if d.Ping != nil {
			ctx, cancel := context.WithTimeout(req.Context(), 2*time.Second)
			defer cancel()
			if err := d.Ping(ctx); err != nil {
				respondError(w, http.StatusServiceUnavailable, "UNAVAILABLE", "database is unavailable")
				return
			}
		}
		w.WriteHeader(http.StatusOK)
	})
	r.Post("/dummyLogin", d.AuthH.DummyLogin)
	r.Post("/register", d.AuthH.Register)
	r.Post("/login", d.AuthH.Login)

	r.Group(func(r chi.Router) {
		r.Use(middleware.Auth(d.JWTSecret))

		r.Get("/rooms/list", d.RoomH.List)
		r.Post("/rooms/create", d.RoomH.Create)

		r.Post("/rooms/{roomId}/schedule/create", d.ScheduleH.Create)

		r.Get("/rooms/{roomId}/slots/list", d.SlotH.List)

		r.Post("/bookings/create", d.BookingH.Create)
		r.Get("/bookings/list", d.BookingH.ListAll)
		r.Get("/bookings/my", d.BookingH.ListMy)
		r.Post("/bookings/{bookingId}/cancel", d.BookingH.Cancel)
	})

	return r
}
