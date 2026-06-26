-- name: CreatePayment :one
INSERT INTO payment.payment (
  user_id, gym_id, amount, currency, payment_type
) VALUES (
  $1, $2, $3, $4, $5
)
RETURNING *;

-- name: GetPayment :one
SELECT * FROM payment.payment
WHERE id = $1 LIMIT 1;

-- name: ListUserPayments :many
SELECT * FROM payment.payment
WHERE user_id = $1
ORDER BY created_at DESC;

-- name: UpdatePaymentStatus :one
UPDATE payment.payment
SET status = $2
WHERE id = $1
RETURNING *;
