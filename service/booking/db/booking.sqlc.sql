-- name: CreateBooking :one
INSERT INTO booking.booking (
  user_id, class_id, status
) VALUES (
  $1, $2, 'confirmed'
)
RETURNING *;

-- name: GetBooking :one
SELECT * FROM booking.booking
WHERE id = $1 LIMIT 1;

-- name: CancelBooking :exec
UPDATE booking.booking
SET status = 'cancelled', cancelled_at = now()
WHERE id = $1;

-- name: ListUserBookings :many
SELECT * FROM booking.booking
WHERE user_id = $1
ORDER BY booked_at DESC;

-- name: ListClassBookings :many
SELECT * FROM booking.booking
WHERE class_id = $1
ORDER BY booked_at DESC;

-- name: CountClassBookings :one
SELECT count(*) FROM booking.booking
WHERE class_id = $1 AND status = 'confirmed';

-- name: CheckDuplicateBooking :one
SELECT EXISTS(
  SELECT 1 FROM booking.booking
  WHERE user_id = $1 AND class_id = $2 AND status = 'confirmed'
) as already_booked;
