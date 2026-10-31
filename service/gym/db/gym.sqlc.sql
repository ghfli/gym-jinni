-- name: CreateGym :one
INSERT INTO gym.gym (
  name, address, phone, email, owner_id
) VALUES (
  $1, $2, $3, $4, $5
)
RETURNING *;

-- name: GetGym :one
SELECT * FROM gym.gym
WHERE id = $1 LIMIT 1;

-- name: ListGyms :many
SELECT * FROM gym.gym
ORDER BY created_at DESC;

-- name: UpdateGym :one
UPDATE gym.gym
SET name = $2, address = $3, phone = $4, email = $5
WHERE id = $1
RETURNING *;

-- name: CreateMembership :one
INSERT INTO gym.membership (
  user_id, gym_id, role
) VALUES (
  $1, $2, $3
)
RETURNING *;

-- name: ListGymMembers :many
SELECT * FROM gym.membership
WHERE gym_id = $1
ORDER BY joined_at DESC;

-- name: GetMembership :one
SELECT * FROM gym.membership
WHERE user_id = $1 AND gym_id = $2 LIMIT 1;

-- name: DeleteMembership :exec
DELETE FROM gym.membership
WHERE user_id = $1 AND gym_id = $2;
