//go:build e2e

package main

import (
	"fmt"
	"reflect"
	"testing"
	"time"
)

func TestE2E_BookingMove(t *testing.T) {
	admin, other := getToken(t, "admin"), getToken(t, "user")
	email := fmt.Sprintf("move-%d@example.com", time.Now().UnixNano())
	code, body := doJSON(t, "POST", "/register", map[string]string{"email": email, "password": "password", "role": "user"})
	if code != 201 {
		t.Fatalf("register: %d %v", code, body)
	}
	code, body = doJSON(t, "POST", "/login", map[string]string{"email": email, "password": "password"})
	if code != 200 {
		t.Fatalf("login: %d %v", code, body)
	}
	owner := body["token"].(string)
	date := time.Now().UTC().AddDate(0, 0, 3).Format("2006-01-02")
	rooms := make([]string, 0, 2)
	slots := make([][]any, 0, 2)
	for i := range 2 {
		code, body = doJSONAuth(t, "POST", "/rooms/create", admin, map[string]string{"name": fmt.Sprintf("move-%d", i)})
		if code != 201 {
			t.Fatalf("room: %d %v", code, body)
		}
		room := body["room"].(map[string]any)["id"].(string)
		rooms = append(rooms, room)
		code, body = doJSONAuth(t, "POST", "/rooms/"+room+"/schedule/create", admin, map[string]any{
			"daysOfWeek": []int{1, 2, 3, 4, 5, 6, 7}, "startTime": "09:00", "endTime": "11:00",
		})
		if code != 201 {
			t.Fatalf("schedule: %d %v", code, body)
		}
		code, body = doJSONAuth(t, "GET", "/rooms/"+room+"/slots/list?date="+date, owner, nil)
		if code != 200 || len(body["slots"].([]any)) != 4 {
			t.Fatalf("slots: %d %v", code, body)
		}
		slots = append(slots, body["slots"].([]any))
	}
	source := slots[0][0].(map[string]any)["id"].(string)
	occupied := slots[1][0].(map[string]any)["id"].(string)
	target := slots[1][1].(map[string]any)["id"].(string)
	obsolete := slots[1][3].(map[string]any)["id"].(string)
	code, body = doJSONAuth(t, "POST", "/bookings/create", owner, map[string]any{"slotId": source, "createConferenceLink": true})
	if code != 201 {
		t.Fatalf("booking: %d %v", code, body)
	}
	original := body["booking"].(map[string]any)
	id := original["id"].(string)
	path := "/bookings/" + id + "/move"
	code, body = doJSONAuth(t, "POST", "/bookings/create", other, map[string]string{"slotId": occupied})
	if code != 201 {
		t.Fatalf("occupied booking: %d %v", code, body)
	}
	checkError := func(wantStatus int, wantCode string, status int, result map[string]any) {
		t.Helper()
		if status != wantStatus || result["error"].(map[string]any)["code"] != wantCode {
			t.Fatalf("expected %d/%s, got %d %v", wantStatus, wantCode, status, result)
		}
	}
	code, body = doJSONAuth(t, "POST", path, other, map[string]string{"slotId": target})
	checkError(403, "FORBIDDEN", code, body)
	code, body = doJSONAuth(t, "POST", path, owner, map[string]string{"slotId": occupied})
	checkError(409, "SLOT_ALREADY_BOOKED", code, body)
	code, body = doJSONAuth(t, "GET", "/bookings/my", owner, nil)
	if code != 200 || len(body["bookings"].([]any)) != 1 || !reflect.DeepEqual(body["bookings"].([]any)[0], original) {
		t.Fatalf("failed move lost original: %d %v", code, body)
	}
	code, body = doJSONAuth(t, "POST", path, owner, map[string]string{"slotId": target})
	if code != 200 {
		t.Fatalf("move: %d %v", code, body)
	}
	want := make(map[string]any, len(original))
	for key, value := range original {
		want[key] = value
	}
	want["slotId"] = target
	if !reflect.DeepEqual(body["booking"], want) {
		t.Fatalf("move changed identity: %v -> %v", original, body)
	}
	for _, response := range parallelRequests(t, "POST", path, owner, map[string]string{"slotId": target}) {
		if response.code != 200 || !reflect.DeepEqual(response.body["booking"], want) {
			t.Fatalf("concurrent repeat: %d %v", response.code, response.body)
		}
	}
	code, body = doJSONAuth(t, "POST", "/bookings/create", other, map[string]string{"slotId": source})
	if code != 201 {
		t.Fatalf("source not released: %d %v", code, body)
	}
	code, body = doJSONAuth(t, "POST", "/rooms/"+rooms[1]+"/schedule/update", admin, map[string]any{
		"daysOfWeek": []int{1, 2, 3, 4, 5, 6, 7}, "startTime": "09:00", "endTime": "10:00",
	})
	if code != 200 {
		t.Fatalf("update: %d %v", code, body)
	}
	code, body = doJSONAuth(t, "POST", path, owner, map[string]string{"slotId": obsolete})
	checkError(409, "BOOKING_MOVE_CONFLICT", code, body)
	code, body = doJSONAuth(t, "POST", "/bookings/"+id+"/cancel", owner, nil)
	want["status"] = "cancelled"
	if code != 200 || !reflect.DeepEqual(body["booking"], want) {
		t.Fatalf("cancel after move: %d %v", code, body)
	}
	code, body = doJSONAuth(t, "POST", path, owner, map[string]string{"slotId": target})
	checkError(409, "BOOKING_MOVE_CONFLICT", code, body)
}
