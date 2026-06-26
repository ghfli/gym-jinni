-- name: CreateActivity :one
INSERT INTO activity.activity (
  user_id, gym_id, activity_type, title, description, duration_minutes, calories, distance_meters
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8
)
RETURNING *;

-- name: GetActivity :one
SELECT * FROM activity.activity
WHERE id = $1 LIMIT 1;

-- name: ListUserActivities :many
SELECT * FROM activity.activity
WHERE user_id = $1
ORDER BY created_at DESC;

-- name: GetUserActivityStats :one
SELECT count(*) as total_count,
       coalesce(sum(duration_minutes),0) as total_duration,
       coalesce(sum(calories),0) as total_calories,
       coalesce(sum(distance_meters),0) as total_distance
FROM activity.activity
WHERE user_id = $1;
