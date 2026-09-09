//go:build e2e

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
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
	req, err := http.NewRequestWithContext(t.Context(), method, baseURL()+path, r)
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
	req, err := http.NewRequestWithContext(t.Context(), method, baseURL()+path, r)
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
	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
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

func TestE2E_ConcurrentGenerationAndBooking(t *testing.T) {
	admin, user := getToken(t, "admin"), getToken(t, "user")
	code, body := doJSONAuth(t, "POST", "/rooms/create", admin, map[string]string{"name": "E2E concurrency"})
	if code != 201 {
		t.Fatalf("room: %d %v", code, body)
	}
	roomID := body["room"].(map[string]any)["id"].(string)
	schedulePath := fmt.Sprintf("/rooms/%s/schedule/create", roomID)
	schedule := map[string]any{"daysOfWeek": []int{1, 2, 3, 4, 5, 6, 7}, "startTime": "9:00", "endTime": "10:00"}
	results := parallelRequests(t, "POST", schedulePath, admin, schedule)
	created := 0
	for _, result := range results {
		switch result.code {
		case 201:
			created++
		case 409:
		default:
			t.Fatalf("schedule race: %#v", result)
		}
	}
	if created != 1 {
		t.Fatalf("schedule winners=%d", created)
	}
	date := time.Now().UTC().AddDate(0, 0, 60).Format("2006-01-02")
	slotPath := fmt.Sprintf("/rooms/%s/slots/list?date=%s", roomID, date)
	results = parallelRequests(t, "GET", slotPath, user, nil)
	var slotID string
	for _, result := range results {
		if result.code != 200 {
			t.Fatalf("slots race: %#v", result)
		}
		slots := result.body["slots"].([]any)
		if len(slots) != 2 {
			t.Fatalf("slots count=%d", len(slots))
		}
		id := slots[0].(map[string]any)["id"].(string)
		if slotID == "" {
			slotID = id
		}
		if slotID != id {
			t.Fatal("concurrent requests returned unstable UUID")
		}
	}
	results = parallelRequests(t, "POST", "/bookings/create", user, map[string]string{"slotId": slotID})
	created = 0
	for _, result := range results {
		switch result.code {
		case 201:
			created++
		case 409:
		default:
			t.Fatalf("booking race: %#v", result)
		}
	}
	if created != 1 {
		t.Fatalf("booking winners=%d", created)
	}
}

type requestResult struct {
	code int
	body map[string]any
	err  error
}

func parallelRequests(t *testing.T, method, path, token string, body any) []requestResult {
	t.Helper()
	encoded, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	const count = 12
	results := make(chan requestResult, count)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for range count {
		wg.Go(func() {
			<-start
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			req, err := http.NewRequestWithContext(ctx, method, baseURL()+path, bytes.NewReader(encoded))
			if err != nil {
				results <- requestResult{err: err}
				return
			}
			req.Header.Set("Authorization", "Bearer "+token)
			req.Header.Set("Content-Type", "application/json")
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				results <- requestResult{err: err}
				return
			}
			defer resp.Body.Close()
			result := requestResult{code: resp.StatusCode}
			result.err = json.NewDecoder(resp.Body).Decode(&result.body)
			results <- result
		})
	}
	close(start)
	wg.Wait()
	close(results)
	all := make([]requestResult, 0, count)
	for result := range results {
		if result.err != nil {
			t.Fatal(result.err)
		}
		all = append(all, result)
	}
	return all
}

func TestE2E_RegistrationOwnershipAndValidation(t *testing.T) {
	admin := getToken(t, "admin")
	var ownerToken, otherToken string
	for i, role := range []string{"user", "user", "admin"} {
		email := fmt.Sprintf("e2e-%d-%d@example.com", time.Now().UnixNano(), i)
		registration := map[string]string{"email": email, "password": "password", "role": role}
		code, body := doJSON(t, "POST", "/register", registration)
		if code != 201 || body["user"].(map[string]any)["role"] != role {
			t.Fatalf("registration: %d %v", code, body)
		}
		code, body = doJSON(t, "POST", "/register", registration)
		if code != 400 || body["error"].(map[string]any)["code"] != "INVALID_REQUEST" {
			t.Fatalf("duplicate email: %d %v", code, body)
		}
		code, body = doJSON(t, "POST", "/login", registration)
		if code != 200 {
			t.Fatalf("login: %d %v", code, body)
		}
		token := body["token"].(string)
		switch i {
		case 0:
			ownerToken = token
		case 1:
			otherToken = token
		case 2:
			admin = token
		}
	}
	code, body := doJSONAuth(t, "POST", "/rooms/create", admin, map[string]string{"name": "E2E ownership"})
	if code != 201 {
		t.Fatalf("registered admin create: %d %v", code, body)
	}
	roomID := body["room"].(map[string]any)["id"].(string)
	code, body = doJSONAuth(t, "POST", fmt.Sprintf("/rooms/%s/schedule/create", roomID), admin, map[string]any{"daysOfWeek": []int{1, 2, 3, 4, 5, 6, 7}, "startTime": "09:00", "endTime": "10:00"})
	if code != 201 {
		t.Fatalf("schedule: %d %v", code, body)
	}
	date := time.Now().UTC().AddDate(0, 0, 1).Format("2006-01-02")
	code, body = doJSONAuth(t, "GET", fmt.Sprintf("/rooms/%s/slots/list?date=%s", roomID, date), ownerToken, nil)
	if code != 200 {
		t.Fatalf("slots: %d %v", code, body)
	}
	slotID := body["slots"].([]any)[0].(map[string]any)["id"].(string)
	code, body = doJSONAuth(t, "POST", "/bookings/create", ownerToken, map[string]string{"slotId": slotID})
	if code != 201 {
		t.Fatalf("booking: %d %v", code, body)
	}
	bookingID := body["booking"].(map[string]any)["id"].(string)
	cancelPath := fmt.Sprintf("/bookings/%s/cancel", bookingID)
	if code, _ := doJSONAuth(t, "POST", cancelPath, otherToken, nil); code != 403 {
		t.Fatalf("nonowner cancellation=%d", code)
	}
	for range 2 {
		code, body := doJSONAuth(t, "POST", cancelPath, ownerToken, nil)
		if code != 200 || body["booking"].(map[string]any)["status"] != "cancelled" {
			t.Fatalf("cancel: %d %v", code, body)
		}
	}
	code, body = doJSONAuth(t, "GET", "/bookings/my", ownerToken, nil)
	if code != 200 {
		t.Fatalf("my: %d %v", code, body)
	}
	my := body["bookings"].([]any)
	if len(my) != 1 || my[0].(map[string]any)["status"] != "cancelled" {
		t.Fatalf("cancelled future booking missing: %v", body)
	}
	for _, tt := range []struct {
		method, path, token string
		body                any
	}{
		{"POST", "/rooms/create", admin, map[string]any{"name": "overflow", "capacity": 2147483648}},
		{"POST", "/rooms/invalid/schedule/create", admin, map[string]any{}}, {"GET", "/rooms/invalid/slots/list?date=" + date, ownerToken, nil},
		{"POST", "/bookings/create", ownerToken, map[string]string{"slotId": "invalid"}}, {"POST", "/bookings/invalid/cancel", ownerToken, nil},
		{"GET", "/bookings/list?page=abc", admin, nil}, {"GET", "/bookings/list?page=0", admin, nil}, {"GET", "/bookings/list?pageSize=101", admin, nil}, {"GET", "/bookings/list?page=9223372036854775807&pageSize=100", admin, nil},
		{"POST", "/register", "", map[string]string{"email": "invalid", "password": "pass", "role": "user"}},
		{"POST", "/register", "", map[string]string{"email": "valid@example.com", "password": "pass"}},
		{"POST", "/register", "", map[string]string{"email": "valid@example.com", "password": strings.Repeat("x", 73), "role": "user"}},
	} {
		code, body := doJSONAuth(t, tt.method, tt.path, tt.token, tt.body)
		if code != 400 {
			t.Errorf("%s %s: %d %v", tt.method, tt.path, code, body)
		}
	}
}
