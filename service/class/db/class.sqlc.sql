-- name: GetClass :one
SELECT * FROM "class".class
WHERE id = $1 LIMIT 1;

-- name: ListClass :many
SELECT * FROM "class".class
ORDER BY start_time;

