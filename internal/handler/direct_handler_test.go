package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/DevDashkovsky/room-booking/internal/middleware"
	"github.com/DevDashkovsky/room-booking/internal/service"
)

func ctxWithRole(role string) context.Context {
	ctx := context.Background()
	ctx = context.WithValue(ctx, middleware.RoleKey, role)
	ctx = context.WithValue(ctx, middleware.UserIDKey, "test-user-id")
	return ctx
}

func ctxWithRoleAndChiParam(role, paramName, paramVal string) context.Context {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(paramName, paramVal)
	ctx := context.WithValue(ctxWithRole(role), chi.RouteCtxKey, rctx)
	return ctx
}

func authSvcNilRepo() *service.AuthService {
	return service.NewAuthService(nil, "secret")
}

func TestAuthHandler_Register_InvalidJSON(t *testing.T) {
	h := NewAuthHandler("secret", nil)
	r := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(`not json`))
	w := httptest.NewRecorder()
	h.Register(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestAuthHandler_Register_EmptyEmail(t *testing.T) {
	h := NewAuthHandler("secret", authSvcNilRepo())
	r := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(`{"email":"","password":"pass"}`))
	w := httptest.NewRecorder()
	h.Register(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestAuthHandler_Login_InvalidJSON(t *testing.T) {
	h := NewAuthHandler("secret", nil)
	r := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(`not json`))
	w := httptest.NewRecorder()
	h.Login(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestAuthHandler_Login_EmptyCredentials(t *testing.T) {
	h := NewAuthHandler("secret", authSvcNilRepo())
	r := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(`{"email":"","password":""}`))
	w := httptest.NewRecorder()
	h.Login(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

func TestBookingHandler_Create_InvalidJSON(t *testing.T) {
	h := NewBookingHandler(nil)
	r := httptest.NewRequest(http.MethodPost, "/bookings/create", strings.NewReader(`not json`))
	r = r.WithContext(ctxWithRole("user"))
	w := httptest.NewRecorder()
	h.Create(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestBookingHandler_Create_EmptySlotId(t *testing.T) {
	h := NewBookingHandler(nil)
	r := httptest.NewRequest(http.MethodPost, "/bookings/create", strings.NewReader(`{"slotId":""}`))
	r = r.WithContext(ctxWithRole("user"))
	w := httptest.NewRecorder()
	h.Create(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestBookingHandler_ListAll_UserForbidden(t *testing.T) {
	h := NewBookingHandler(nil)
	r := httptest.NewRequest(http.MethodGet, "/bookings/list", nil)
	r = r.WithContext(ctxWithRole("user"))
	w := httptest.NewRecorder()
	h.ListAll(w, r)
	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403", w.Code)
	}
}

func TestBookingHandler_ListMy_AdminForbidden(t *testing.T) {
	h := NewBookingHandler(nil)
	r := httptest.NewRequest(http.MethodGet, "/bookings/my", nil)
	r = r.WithContext(ctxWithRole("admin"))
	w := httptest.NewRecorder()
	h.ListMy(w, r)
	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403", w.Code)
	}
}

func TestBookingHandler_Cancel_AdminForbidden(t *testing.T) {
	h := NewBookingHandler(nil)
	r := httptest.NewRequest(http.MethodPost, "/bookings/cancel", nil)
	r = r.WithContext(ctxWithRole("admin"))
	w := httptest.NewRecorder()
	h.Cancel(w, r)
	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403", w.Code)
	}
}

func TestBookingHandler_Cancel_EmptyBookingId(t *testing.T) {
	h := NewBookingHandler(nil)
	r := httptest.NewRequest(http.MethodPost, "/bookings/cancel", nil)
	r = r.WithContext(ctxWithRole("user")) // no chi context → bookingId == ""
	w := httptest.NewRecorder()
	h.Cancel(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestRoomHandler_Create_InvalidJSON(t *testing.T) {
	h := NewRoomHandler(nil)
	r := httptest.NewRequest(http.MethodPost, "/rooms/create", strings.NewReader(`not json`))
	r = r.WithContext(ctxWithRole("admin"))
	w := httptest.NewRecorder()
	h.Create(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestScheduleHandler_Create_AdminMissingRoomId(t *testing.T) {
	h := NewScheduleHandler(nil)
	r := httptest.NewRequest(http.MethodPost, "/rooms//schedule/create", strings.NewReader(`{}`))
	r = r.WithContext(ctxWithRole("admin")) // no chi context → roomId == ""
	w := httptest.NewRecorder()
	h.Create(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestScheduleHandler_Create_AdminInvalidJSON(t *testing.T) {
	h := NewScheduleHandler(nil)
	r := httptest.NewRequest(http.MethodPost, "/rooms/room-1/schedule/create", strings.NewReader(`not json`))
	r = r.WithContext(ctxWithRoleAndChiParam("admin", "roomId", "room-1"))
	w := httptest.NewRecorder()
	h.Create(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestSlotHandler_List_MissingRoomId(t *testing.T) {
	h := NewSlotHandler(nil)
	r := httptest.NewRequest(http.MethodGet, "/rooms//slots/list?date=2024-01-01", nil)
	r = r.WithContext(ctxWithRole("user")) // no chi context → roomId == ""
	w := httptest.NewRecorder()
	h.List(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}
