-- name: GetClass :one
SELECT * FROM "class".class
WHERE id = $1 LIMIT 1;

-- name: ListClass :many
SELECT * FROM "class".class
ORDER BY start_time;

-- name: CreateClass :one
INSERT INTO "class".class (
  created_by, description, start_time, end_time, min_hdcnt, max_hdcnt
) VALUES (
  $1, $2, $3, $4, $5, $6
)
RETURNING *;

-- name: UpdateClass :exec
UPDATE "class".class
SET description = $2, start_time = $3, end_time = $4, min_hdcnt = $5, max_hdcnt = $6
WHERE id = $1;

-- name: DeleteClass :exec
DELETE FROM "class".class
WHERE id = $1;

