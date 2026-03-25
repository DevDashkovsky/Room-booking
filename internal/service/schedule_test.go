package service

import (
	"testing"
	"time"

	"github.com/DevDashkovsky/room-booking/internal/domain"
)

func TestValidateSchedule(t *testing.T) {
	tests := []struct {
		name    string
		s       *domain.Schedule
		wantErr bool
	}{
		{
			name:    "valid",
			s:       &domain.Schedule{DaysOfWeek: []int{1, 3, 5}, StartTime: "09:00", EndTime: "18:00"},
			wantErr: false,
		},
		{
			name:    "empty days",
			s:       &domain.Schedule{DaysOfWeek: []int{}, StartTime: "09:00", EndTime: "18:00"},
			wantErr: true,
		},
		{
			name:    "day 0",
			s:       &domain.Schedule{DaysOfWeek: []int{0}, StartTime: "09:00", EndTime: "18:00"},
			wantErr: true,
		},
		{
			name:    "day 8",
			s:       &domain.Schedule{DaysOfWeek: []int{8}, StartTime: "09:00", EndTime: "18:00"},
			wantErr: true,
		},
		{
			name:    "start >= end",
			s:       &domain.Schedule{DaysOfWeek: []int{1}, StartTime: "18:00", EndTime: "09:00"},
			wantErr: true,
		},
		{
			name:    "same time",
			s:       &domain.Schedule{DaysOfWeek: []int{1}, StartTime: "09:00", EndTime: "09:00"},
			wantErr: true,
		},
		{
			name:    "bad start format",
			s:       &domain.Schedule{DaysOfWeek: []int{1}, StartTime: "9", EndTime: "18:00"},
			wantErr: true,
		},
		{
			name:    "bad end format",
			s:       &domain.Schedule{DaysOfWeek: []int{1}, StartTime: "09:00", EndTime: "abc"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSchedule(tt.s)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateSchedule() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGenerateSlots(t *testing.T) {
	schedule := &domain.Schedule{
		RoomID:     "room-1",
		DaysOfWeek: []int{1}, // Monday
		StartTime:  "09:00",
		EndTime:    "11:00",
	}

	from := time.Date(2024, 6, 10, 0, 0, 0, 0, time.UTC)
	slots := generateSlots(schedule, from, 7)

	if len(slots) != 4 {
		t.Fatalf("got %d slots, want 4", len(slots))
	}

	if slots[0].Start.Hour() != 9 || slots[0].Start.Minute() != 0 {
		t.Errorf("first slot start = %v, want 09:00", slots[0].Start)
	}
	if slots[3].End.Hour() != 11 || slots[3].End.Minute() != 0 {
		t.Errorf("last slot end = %v, want 11:00", slots[3].End)
	}
}

func TestGenerateSlotsMultipleDays(t *testing.T) {
	schedule := &domain.Schedule{
		RoomID:     "room-1",
		DaysOfWeek: []int{1, 3, 5}, // Mon, Wed, Fri
		StartTime:  "10:00",
		EndTime:    "11:00",
	}

	from := time.Date(2024, 6, 10, 0, 0, 0, 0, time.UTC)
	slots := generateSlots(schedule, from, 7)

	if len(slots) != 6 {
		t.Fatalf("got %d slots, want 6", len(slots))
	}
}

func TestGenerateSlotsSunday(t *testing.T) {
	schedule := &domain.Schedule{
		RoomID:     "room-1",
		DaysOfWeek: []int{7},
		StartTime:  "10:00",
		EndTime:    "10:30",
	}

	from := time.Date(2024, 6, 9, 0, 0, 0, 0, time.UTC)
	slots := generateSlots(schedule, from, 1)

	if len(slots) != 1 {
		t.Fatalf("got %d slots, want 1", len(slots))
	}
}

func TestParseHHMM(t *testing.T) {
	tests := []struct {
		input string
		h, m  int
		ok    bool
	}{
		{"09:00", 9, 0, true},
		{"23:59", 23, 59, true},
		{"00:00", 0, 0, true},
		{"24:00", 0, 0, false},
		{"09:60", 0, 0, false},
		{"abc", 0, 0, false},
		{"9", 0, 0, false},
		{"", 0, 0, false},
		{"-1:00", 0, 0, false},
		{"09:-1", 0, 0, false},
		{"a:b", 0, 0, false},
		{"12:00:00", 0, 0, false},
	}

	for _, tt := range tests {
		h, m, ok := parseHHMM(tt.input)
		if ok != tt.ok || (ok && (h != tt.h || m != tt.m)) {
			t.Errorf("parseHHMM(%q) = %d,%d,%v want %d,%d,%v", tt.input, h, m, ok, tt.h, tt.m, tt.ok)
		}
	}
}

func TestValidateSchedule_NegativeMinutes(t *testing.T) {
	s := &domain.Schedule{DaysOfWeek: []int{1}, StartTime: "09:-5", EndTime: "18:00"}
	if err := validateSchedule(s); err == nil {
		t.Error("expected error for negative minutes")
	}
}

func TestGenerateSlots_NoMatchingDays(t *testing.T) {
	schedule := &domain.Schedule{
		RoomID:     "room-1",
		DaysOfWeek: []int{6}, // Saturday
		StartTime:  "09:00",
		EndTime:    "10:00",
	}
	// 2024-06-10 is Monday, check 5 days Mon-Fri, no Saturday
	from := time.Date(2024, 6, 10, 0, 0, 0, 0, time.UTC)
	slots := generateSlots(schedule, from, 5)
	if len(slots) != 0 {
		t.Errorf("got %d slots, want 0", len(slots))
	}
}

func TestGenerateSlots_AllWeekDays(t *testing.T) {
	schedule := &domain.Schedule{
		RoomID:     "room-1",
		DaysOfWeek: []int{1, 2, 3, 4, 5, 6, 7},
		StartTime:  "10:00",
		EndTime:    "10:30",
	}
	from := time.Date(2024, 6, 10, 0, 0, 0, 0, time.UTC)
	slots := generateSlots(schedule, from, 7)
	if len(slots) != 7 {
		t.Fatalf("got %d slots, want 7 (one per day)", len(slots))
	}
}

func TestValidateSchedule_Day7Sunday(t *testing.T) {
	s := &domain.Schedule{DaysOfWeek: []int{7}, StartTime: "09:00", EndTime: "10:00"}
	if err := validateSchedule(s); err != nil {
		t.Errorf("day 7 (Sunday) should be valid: %v", err)
	}
}
