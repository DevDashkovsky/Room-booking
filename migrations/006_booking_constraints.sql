-- +goose Up
CREATE UNIQUE INDEX IF NOT EXISTS uniq_active_booking_per_slot ON bookings (slot_id) WHERE status = 'active';

DROP INDEX IF EXISTS idx_slots_room_date;

-- +goose Down
CREATE INDEX IF NOT EXISTS idx_slots_room_date ON slots (room_id, start_at);
DROP INDEX IF EXISTS uniq_active_booking_per_slot;
