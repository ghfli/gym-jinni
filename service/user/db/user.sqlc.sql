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

-- name: CreateSession :one
INSERT INTO "user".session (
  id, user_id, refresh_tkn, user_agent, client_ip, expires_at, created_at
) VALUES (
  $1, $2, $3, $4, $5, $6, $7
)
RETURNING *;

-- name: GetSession :one
SELECT * FROM "user".session
WHERE id = $1 LIMIT 1;

-- name: BlockSession :exec
UPDATE "user".session
SET blocked = true
WHERE id = $1;

-- name: CreateTrainerProfile :one
INSERT INTO "user".trainer_profile (
  user_id, bio, specializations, certifications, hourly_rate
) VALUES (
  $1, $2, $3, $4, $5
)
RETURNING *;

-- name: GetTrainerProfile :one
SELECT * FROM "user".trainer_profile
WHERE user_id = $1 LIMIT 1;

-- name: UpdateTrainerProfile :one
UPDATE "user".trainer_profile
SET bio = $2, specializations = $3, certifications = $4, hourly_rate = $5
WHERE user_id = $1
RETURNING *;

-- name: ListTrainerProfiles :many
SELECT * FROM "user".trainer_profile
ORDER BY created_at DESC;

-- name: DeleteTrainerProfile :exec
DELETE FROM "user".trainer_profile
WHERE user_id = $1;
