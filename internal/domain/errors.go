package domain

import "errors"

var (
	ErrInvalidRequest    = errors.New("INVALID_REQUEST")
	ErrUnauthorized      = errors.New("UNAUTHORIZED")
	ErrForbidden         = errors.New("FORBIDDEN")
	ErrNotFound          = errors.New("NOT_FOUND")
	ErrRoomNotFound      = errors.New("ROOM_NOT_FOUND")
	ErrSlotNotFound      = errors.New("SLOT_NOT_FOUND")
	ErrSlotAlreadyBooked = errors.New("SLOT_ALREADY_BOOKED")
	ErrBookingNotFound   = errors.New("BOOKING_NOT_FOUND")
	ErrScheduleExists    = errors.New("SCHEDULE_EXISTS")
	ErrInternalError     = errors.New("INTERNAL_ERROR")
)
