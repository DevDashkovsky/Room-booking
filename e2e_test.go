//go:build e2e

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"
)

func baseURL() string {
	if v := os.Getenv("BASE_URL"); v != "" {
		return v
	}
	return "http://localhost:8080"
}

func doJSON(t *testing.T, method, path string, body any) (int, map[string]any) {
	t.Helper()
	var r io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		r = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, baseURL()+path, r)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return execReq(t, req)
}

func doJSONAuth(t *testing.T, method, path, token string, body any) (int, map[string]any) {
	t.Helper()
	var r io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		r = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, baseURL()+path, r)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Authorization", "Bearer "+token)
	return execReq(t, req)
}

func execReq(t *testing.T, req *http.Request) (int, map[string]any) {
	t.Helper()
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)
	return resp.StatusCode, result
}

func getToken(t *testing.T, role string) string {
	t.Helper()
	code, body := doJSON(t, "POST", "/dummyLogin", map[string]string{"role": role})
	if code != 200 {
		t.Fatalf("dummyLogin %s: status %d", role, code)
	}
	token, _ := body["token"].(string)
	if token == "" {
		t.Fatal("empty token")
	}
	return token
}

func nextWeekday() string {
	d := time.Now().UTC().AddDate(0, 0, 1)
	for d.Weekday() == time.Saturday || d.Weekday() == time.Sunday {
		d = d.AddDate(0, 0, 1)
	}
	return d.Format("2006-01-02")
}

func TestE2E_CreateRoomScheduleBooking(t *testing.T) {
	adminToken := getToken(t, "admin")
	userToken := getToken(t, "user")

	code, body := doJSONAuth(t, "POST", "/rooms/create", adminToken, map[string]string{"name": "E2E Room 1"})
	if code != 201 {
		t.Fatalf("create room: status %d, body %v", code, body)
	}
	room := body["room"].(map[string]any)
	roomID := room["id"].(string)
	t.Logf("Room created: %s", roomID)

	code, body = doJSONAuth(t, "POST", fmt.Sprintf("/rooms/%s/schedule/create", roomID), adminToken, map[string]any{
		"daysOfWeek": []int{1, 2, 3, 4, 5},
		"startTime":  "09:00",
		"endTime":    "18:00",
	})
	if code != 201 {
		t.Fatalf("create schedule: status %d, body %v", code, body)
	}
	t.Log("Schedule created")

	date := nextWeekday()
	code, body = doJSONAuth(t, "GET", fmt.Sprintf("/rooms/%s/slots/list?date=%s", roomID, date), userToken, nil)
	if code != 200 {
		t.Fatalf("list slots: status %d, body %v", code, body)
	}
	slots := body["slots"].([]any)
	if len(slots) == 0 {
		t.Fatal("no slots returned")
	}
	t.Logf("Slots on %s: %d", date, len(slots))

	if len(slots) != 18 {
		t.Errorf("expected 18 slots, got %d", len(slots))
	}

	slotID := slots[0].(map[string]any)["id"].(string)
	code, body = doJSONAuth(t, "POST", "/bookings/create", userToken, map[string]string{"slotId": slotID})
	if code != 201 {
		t.Fatalf("create booking: status %d, body %v", code, body)
	}
	booking := body["booking"].(map[string]any)
	if booking["status"] != "active" {
		t.Errorf("booking status = %v, want active", booking["status"])
	}
	t.Logf("Booking created: %s", booking["id"])

	code, body = doJSONAuth(t, "GET", fmt.Sprintf("/rooms/%s/slots/list?date=%s", roomID, date), userToken, nil)
	if code != 200 {
		t.Fatalf("list slots after booking: status %d", code)
	}
	slotsAfter := body["slots"].([]any)
	if len(slotsAfter) != len(slots)-1 {
		t.Errorf("expected %d slots after booking, got %d", len(slots)-1, len(slotsAfter))
	}

	code, _ = doJSONAuth(t, "POST", "/bookings/create", userToken, map[string]string{"slotId": slotID})
	if code != 409 {
		t.Errorf("double booking: status %d, want 409", code)
	}

	code, _ = doJSONAuth(t, "POST", "/bookings/create", adminToken, map[string]string{"slotId": slotID})
	if code != 403 {
		t.Errorf("admin booking: status %d, want 403", code)
	}

	code, body = doJSONAuth(t, "GET", "/bookings/my", userToken, nil)
	if code != 200 {
		t.Fatalf("bookings/my: status %d", code)
	}
	myBookings := body["bookings"].([]any)
	if len(myBookings) == 0 {
		t.Error("bookings/my returned empty list")
	}

	code, body = doJSONAuth(t, "GET", "/bookings/list", adminToken, nil)
	if code != 200 {
		t.Fatalf("bookings/list: status %d", code)
	}
	if body["pagination"] == nil {
		t.Error("bookings/list missing pagination")
	}
}

func TestE2E_CancelBooking(t *testing.T) {
	adminToken := getToken(t, "admin")
	userToken := getToken(t, "user")

	code, body := doJSONAuth(t, "POST", "/rooms/create", adminToken, map[string]string{"name": "E2E Room 2"})
	if code != 201 {
		t.Fatalf("create room: status %d", code)
	}
	roomID := body["room"].(map[string]any)["id"].(string)

	code, _ = doJSONAuth(t, "POST", fmt.Sprintf("/rooms/%s/schedule/create", roomID), adminToken, map[string]any{
		"daysOfWeek": []int{1, 2, 3, 4, 5},
		"startTime":  "10:00",
		"endTime":    "12:00",
	})
	if code != 201 {
		t.Fatalf("create schedule: status %d", code)
	}

	date := nextWeekday()
	code, body = doJSONAuth(t, "GET", fmt.Sprintf("/rooms/%s/slots/list?date=%s", roomID, date), userToken, nil)
	if code != 200 {
		t.Fatalf("list slots: status %d", code)
	}
	slots := body["slots"].([]any)
	if len(slots) == 0 {
		t.Fatal("no slots")
	}
	slotsBefore := len(slots)

	slotID := slots[0].(map[string]any)["id"].(string)
	code, body = doJSONAuth(t, "POST", "/bookings/create", userToken, map[string]string{"slotId": slotID})
	if code != 201 {
		t.Fatalf("create booking: status %d", code)
	}
	bookingID := body["booking"].(map[string]any)["id"].(string)
	t.Logf("Booking to cancel: %s", bookingID)

	code, body = doJSONAuth(t, "POST", fmt.Sprintf("/bookings/%s/cancel", bookingID), userToken, nil)
	if code != 200 {
		t.Fatalf("cancel: status %d, body %v", code, body)
	}
	if body["booking"].(map[string]any)["status"] != "cancelled" {
		t.Errorf("status = %v, want cancelled", body["booking"].(map[string]any)["status"])
	}

	code, body = doJSONAuth(t, "POST", fmt.Sprintf("/bookings/%s/cancel", bookingID), userToken, nil)
	if code != 200 {
		t.Errorf("repeat cancel: status %d, want 200", code)
	}
	if body["booking"].(map[string]any)["status"] != "cancelled" {
		t.Errorf("repeat cancel status = %v, want cancelled", body["booking"].(map[string]any)["status"])
	}

	code, body = doJSONAuth(t, "GET", fmt.Sprintf("/rooms/%s/slots/list?date=%s", roomID, date), userToken, nil)
	if code != 200 {
		t.Fatalf("list slots after cancel: status %d", code)
	}
	slotsAfter := len(body["slots"].([]any))
	if slotsAfter != slotsBefore {
		t.Errorf("slots after cancel = %d, want %d (slot should be available again)", slotsAfter, slotsBefore)
	}

	code, _ = doJSONAuth(t, "POST", "/bookings/00000000-0000-0000-0000-000000000099/cancel", userToken, nil)
	if code != 404 {
		t.Errorf("cancel nonexistent: status %d, want 404", code)
	}

	code, _ = doJSONAuth(t, "POST", fmt.Sprintf("/bookings/%s/cancel", bookingID), adminToken, nil)
	if code != 403 {
		t.Errorf("admin cancel: status %d, want 403", code)
	}
}
