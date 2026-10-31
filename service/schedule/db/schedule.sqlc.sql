-- name: CreateSchedule :one
INSERT INTO schedule.schedule (
  gym_id, trainer_id, class_id, recurrence_rule, start_date, end_date
) VALUES (
  $1, $2, $3, $4, $5, $6
)
RETURNING *;

-- name: GetSchedule :one
SELECT * FROM schedule.schedule
WHERE id = $1 LIMIT 1;

-- name: ListSchedules :many
SELECT * FROM schedule.schedule
ORDER BY start_date DESC;

-- name: ListSchedulesByGym :many
SELECT * FROM schedule.schedule
WHERE gym_id = $1
ORDER BY start_date DESC;

-- name: UpdateSchedule :one
UPDATE schedule.schedule
SET recurrence_rule = $2, start_date = $3, end_date = $4
WHERE id = $1
RETURNING *;

-- name: DeleteSchedule :exec
DELETE FROM schedule.schedule
WHERE id = $1;
