package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	myjwt "github.com/DevDashkovsky/room-booking/internal/jwt"
)

func newRouterWithAuth() http.Handler {
	return NewRouter(Deps{
		JWTSecret: "secret",
		AuthH:     NewAuthHandler("secret"),
	})
}

func adminToken() string {
	t, _ := myjwt.GenerateToken("00000000-0000-0000-0000-000000000001", "admin", "secret")
	return t
}

func userToken() string {
	t, _ := myjwt.GenerateToken("00000000-0000-0000-0000-000000000002", "user", "secret")
	return t
}

func TestRoomCreate_NoAuth(t *testing.T) {
	r := newRouterWithAuth()
	req := httptest.NewRequest(http.MethodPost, "/rooms/create", strings.NewReader(`{"name":"test"}`))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

func TestRoomCreate_UserForbidden(t *testing.T) {
	r := newRouterWithAuth()
	req := httptest.NewRequest(http.MethodPost, "/rooms/create", strings.NewReader(`{"name":"test"}`))
	req.Header.Set("Authorization", "Bearer "+userToken())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403", w.Code)
	}
}

func TestRoomList_NoAuth(t *testing.T) {
	r := newRouterWithAuth()
	req := httptest.NewRequest(http.MethodGet, "/rooms/list", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

func TestDummyLogin_ViaRouter(t *testing.T) {
	r := newRouterWithAuth()
	req := httptest.NewRequest(http.MethodPost, "/dummyLogin", strings.NewReader(`{"role":"admin"}`))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}

	var body map[string]string
	json.NewDecoder(w.Body).Decode(&body)
	if body["token"] == "" {
		t.Error("token is empty")
	}
}

func TestBookingCreate_AdminForbidden(t *testing.T) {
	r := newRouterWithAuth()
	req := httptest.NewRequest(http.MethodPost, "/bookings/create", strings.NewReader(`{"slotId":"some-id"}`))
	req.Header.Set("Authorization", "Bearer "+adminToken())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403", w.Code)
	}
}

func TestBookingListAll_UserForbidden(t *testing.T) {
	r := newRouterWithAuth()
	req := httptest.NewRequest(http.MethodGet, "/bookings/list", nil)
	req.Header.Set("Authorization", "Bearer "+userToken())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403", w.Code)
	}
}

func TestBookingMy_AdminForbidden(t *testing.T) {
	r := newRouterWithAuth()
	req := httptest.NewRequest(http.MethodGet, "/bookings/my", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403", w.Code)
	}
}

func TestScheduleCreate_UserForbidden(t *testing.T) {
	r := newRouterWithAuth()
	req := httptest.NewRequest(http.MethodPost, "/rooms/some-id/schedule/create", strings.NewReader(`{}`))
	req.Header.Set("Authorization", "Bearer "+userToken())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403", w.Code)
	}
}

func TestSlotList_NoDate(t *testing.T) {
	r := newRouterWithAuth()
	req := httptest.NewRequest(http.MethodGet, "/rooms/some-id/slots/list", nil)
	req.Header.Set("Authorization", "Bearer "+userToken())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestSlotList_BadDate(t *testing.T) {
	r := newRouterWithAuth()
	req := httptest.NewRequest(http.MethodGet, "/rooms/some-id/slots/list?date=not-a-date", nil)
	req.Header.Set("Authorization", "Bearer "+userToken())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestBookingCancel_AdminForbidden(t *testing.T) {
	r := newRouterWithAuth()
	req := httptest.NewRequest(http.MethodPost, "/bookings/some-id/cancel", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403", w.Code)
	}
}

func TestBookingCancel_NoAuth(t *testing.T) {
	r := newRouterWithAuth()
	req := httptest.NewRequest(http.MethodPost, "/bookings/some-id/cancel", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

func TestBookingCreate_NoAuth(t *testing.T) {
	r := newRouterWithAuth()
	req := httptest.NewRequest(http.MethodPost, "/bookings/create", strings.NewReader(`{}`))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

func TestBookingMy_NoAuth(t *testing.T) {
	r := newRouterWithAuth()
	req := httptest.NewRequest(http.MethodGet, "/bookings/my", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

func TestBookingList_NoAuth(t *testing.T) {
	r := newRouterWithAuth()
	req := httptest.NewRequest(http.MethodGet, "/bookings/list", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

func TestScheduleCreate_NoAuth(t *testing.T) {
	r := newRouterWithAuth()
	req := httptest.NewRequest(http.MethodPost, "/rooms/some-id/schedule/create", strings.NewReader(`{}`))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

func TestSlotList_NoAuth(t *testing.T) {
	r := newRouterWithAuth()
	req := httptest.NewRequest(http.MethodGet, "/rooms/some-id/slots/list?date=2024-01-01", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
}
