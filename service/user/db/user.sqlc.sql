-- name: GetUser :one
SELECT * FROM "user".user
WHERE id = $1 LIMIT 1;

-- name: ListUser :many
SELECT * FROM "user".user
ORDER BY name;

/* use :copyfrom instead of :one if do not need returning */
-- name: CreateUser :one
INSERT INTO "user".user (
  email, phone, name, hashed_passwd
) VALUES (
  $1, $2, $3, $4
)
returning *;

-- name: UpdateUser :exec
UPDATE "user".user
set email = $2, phone = $3, name = $4, hashed_passwd = $5
WHERE id = $1;

-- name: DeleteUser :exec
DELETE FROM "user".user
WHERE id = $1;
