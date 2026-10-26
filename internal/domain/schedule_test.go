package domain

import (
	"testing"
	"time"
)

func TestScheduleAllows(t *testing.T) {
	schedule := &Schedule{RoomID: "room", DaysOfWeek: []int{1}, StartTime: "9:00", EndTime: "10:15"}
	start := time.Date(2026, 10, 26, 9, 0, 0, 0, time.UTC)
	tests := []struct {
		name string
		slot Slot
		want bool
	}{
		{"first", Slot{RoomID: "room", Start: start, End: start.Add(30 * time.Minute)}, true},
		{"offset zone", Slot{RoomID: "room", Start: start.In(time.FixedZone("UTC+7", 7*3600)), End: start.Add(30 * time.Minute)}, true},
		{"last full", Slot{RoomID: "room", Start: start.Add(30 * time.Minute), End: start.Add(time.Hour)}, true},
		{"remainder", Slot{RoomID: "room", Start: start.Add(time.Hour), End: start.Add(90 * time.Minute)}, false},
		{"shift", Slot{RoomID: "room", Start: start.Add(15 * time.Minute), End: start.Add(45 * time.Minute)}, false},
		{"seconds", Slot{RoomID: "room", Start: start.Add(time.Second), End: start.Add(30*time.Minute + time.Second)}, false},
		{"wrong room", Slot{RoomID: "other", Start: start, End: start.Add(30 * time.Minute)}, false},
		{"wrong day", Slot{RoomID: "room", Start: start.AddDate(0, 0, 1), End: start.AddDate(0, 0, 1).Add(30 * time.Minute)}, false},
		{"wrong duration", Slot{RoomID: "room", Start: start, End: start.Add(time.Hour)}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := schedule.Allows(&tt.slot); got != tt.want {
				t.Fatalf("Allows=%v, want %v", got, tt.want)
			}
		})
	}
}
