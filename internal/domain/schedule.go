package domain

import (
	"strconv"
	"strings"
	"time"
)

type Schedule struct {
	ID         string `json:"id,omitempty"`
	RoomID     string `json:"roomId"`
	DaysOfWeek []int  `json:"daysOfWeek"`
	StartTime  string `json:"startTime"`
	EndTime    string `json:"endTime"`
}

func ParseScheduleTime(t string) (int, int, bool) {
	parts := strings.Split(t, ":")
	if len(parts) != 2 || len(parts[0]) < 1 || len(parts[0]) > 2 || len(parts[1]) != 2 {
		return 0, 0, false
	}
	for _, part := range parts {
		for _, c := range part {
			if c < '0' || c > '9' {
				return 0, 0, false
			}
		}
	}
	h, err1 := strconv.Atoi(parts[0])
	m, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil || h < 0 || h > 23 || m < 0 || m > 59 {
		return 0, 0, false
	}
	return h, m, true
}

func GenerateSlots(schedule *Schedule, from time.Time, days int) []Slot {
	from = from.UTC()
	startH, startM, _ := ParseScheduleTime(schedule.StartTime)
	endH, endM, _ := ParseScheduleTime(schedule.EndTime)

	daySet := make(map[time.Weekday]bool)
	for _, d := range schedule.DaysOfWeek {
		if d == 7 {
			daySet[time.Sunday] = true
		} else {
			daySet[time.Weekday(d)] = true
		}
	}

	startOfDay := time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, time.UTC)

	var slots []Slot
	for i := 0; i < days; i++ {
		day := startOfDay.AddDate(0, 0, i)
		if !daySet[day.Weekday()] {
			continue
		}

		slotStart := time.Date(day.Year(), day.Month(), day.Day(), startH, startM, 0, 0, time.UTC)
		slotEndLimit := time.Date(day.Year(), day.Month(), day.Day(), endH, endM, 0, 0, time.UTC)

		for slotStart.Add(30*time.Minute).Before(slotEndLimit) || slotStart.Add(30*time.Minute).Equal(slotEndLimit) {
			slots = append(slots, Slot{
				RoomID: schedule.RoomID,
				Start:  slotStart,
				End:    slotStart.Add(30 * time.Minute),
			})
			slotStart = slotStart.Add(30 * time.Minute)
		}
	}

	return slots
}

// Allows reports whether slot is one complete slot of this schedule's UTC grid.
func (s *Schedule) Allows(slot *Slot) bool {
	if s.RoomID != slot.RoomID || slot.End.Sub(slot.Start) != 30*time.Minute {
		return false
	}
	start := slot.Start.UTC()
	if start.Second() != 0 || start.Nanosecond() != 0 {
		return false
	}
	weekday := int(start.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	found := false
	for _, day := range s.DaysOfWeek {
		if day == weekday {
			found = true
			break
		}
	}
	if !found {
		return false
	}
	sh, sm, ok := ParseScheduleTime(s.StartTime)
	if !ok {
		return false
	}
	eh, em, ok := ParseScheduleTime(s.EndTime)
	if !ok {
		return false
	}
	minute := start.Hour()*60 + start.Minute()
	return minute >= sh*60+sm && minute+30 <= eh*60+em && (minute-sh*60-sm)%30 == 0
}
