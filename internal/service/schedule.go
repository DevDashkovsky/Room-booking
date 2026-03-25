package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/DevDashkovsky/room-booking/internal/domain"
	"github.com/DevDashkovsky/room-booking/internal/repository"
)

type ScheduleService struct {
	scheduleRepo *repository.ScheduleRepository
	roomRepo     *repository.RoomRepository
	slotRepo     *repository.SlotRepository
}

func NewScheduleService(
	scheduleRepo *repository.ScheduleRepository,
	roomRepo *repository.RoomRepository,
	slotRepo *repository.SlotRepository,
) *ScheduleService {
	return &ScheduleService{
		scheduleRepo: scheduleRepo,
		roomRepo:     roomRepo,
		slotRepo:     slotRepo,
	}
}

func (s *ScheduleService) Create(ctx context.Context, schedule *domain.Schedule) error {
	if err := validateSchedule(schedule); err != nil {
		return err
	}

	room, err := s.roomRepo.GetByID(ctx, schedule.RoomID)
	if err != nil {
		return fmt.Errorf("get room: %w", err)
	}
	if room == nil {
		return domain.ErrRoomNotFound
	}

	exists, err := s.scheduleRepo.Exists(ctx, schedule.RoomID)
	if err != nil {
		return fmt.Errorf("check schedule exists: %w", err)
	}
	if exists {
		return domain.ErrScheduleExists
	}

	if err := s.scheduleRepo.Create(ctx, schedule); err != nil {
		return fmt.Errorf("create schedule: %w", err)
	}

	slots := generateSlots(schedule, time.Now().UTC(), 30)
	if err := s.slotRepo.BulkCreate(ctx, slots); err != nil {
		return fmt.Errorf("create slots: %w", err)
	}

	return nil
}

func validateSchedule(s *domain.Schedule) error {
	if len(s.DaysOfWeek) == 0 {
		return domain.ErrInvalidRequest
	}
	for _, d := range s.DaysOfWeek {
		if d < 1 || d > 7 {
			return domain.ErrInvalidRequest
		}
	}

	startH, startM, ok1 := parseHHMM(s.StartTime)
	endH, endM, ok2 := parseHHMM(s.EndTime)
	if !ok1 || !ok2 {
		return domain.ErrInvalidRequest
	}

	if startH*60+startM >= endH*60+endM {
		return domain.ErrInvalidRequest
	}

	return nil
}

func parseHHMM(t string) (int, int, bool) {
	parts := strings.Split(t, ":")
	if len(parts) != 2 {
		return 0, 0, false
	}
	h, err1 := strconv.Atoi(parts[0])
	m, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil || h < 0 || h > 23 || m < 0 || m > 59 {
		return 0, 0, false
	}
	return h, m, true
}

func generateSlots(schedule *domain.Schedule, from time.Time, days int) []domain.Slot {
	startH, startM, _ := parseHHMM(schedule.StartTime)
	endH, endM, _ := parseHHMM(schedule.EndTime)

	daySet := make(map[time.Weekday]bool)
	for _, d := range schedule.DaysOfWeek {
		if d == 7 {
			daySet[time.Sunday] = true
		} else {
			daySet[time.Weekday(d)] = true
		}
	}

	startOfDay := time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, time.UTC)

	var slots []domain.Slot
	for i := 0; i < days; i++ {
		day := startOfDay.AddDate(0, 0, i)
		if !daySet[day.Weekday()] {
			continue
		}

		slotStart := time.Date(day.Year(), day.Month(), day.Day(), startH, startM, 0, 0, time.UTC)
		slotEndLimit := time.Date(day.Year(), day.Month(), day.Day(), endH, endM, 0, 0, time.UTC)

		for slotStart.Add(30*time.Minute).Before(slotEndLimit) || slotStart.Add(30*time.Minute).Equal(slotEndLimit) {
			slots = append(slots, domain.Slot{
				RoomID: schedule.RoomID,
				Start:  slotStart,
				End:    slotStart.Add(30 * time.Minute),
			})
			slotStart = slotStart.Add(30 * time.Minute)
		}
	}

	return slots
}
