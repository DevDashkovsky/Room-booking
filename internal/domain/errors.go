package domain

import "errors"

var (
	ErrInvalidRequest    = errors.New("INVALID_REQUEST")
	ErrForbidden         = errors.New("FORBIDDEN")
	ErrRoomNotFound      = errors.New("ROOM_NOT_FOUND")
	ErrSlotNotFound      = errors.New("SLOT_NOT_FOUND")
	ErrSlotAlreadyBooked = errors.New("SLOT_ALREADY_BOOKED")
	ErrBookingNotFound   = errors.New("BOOKING_NOT_FOUND")
	ErrScheduleExists    = errors.New("SCHEDULE_EXISTS")
	ErrUnauthorized      = errors.New("UNAUTHORIZED")
)
