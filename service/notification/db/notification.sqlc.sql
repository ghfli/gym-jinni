-- name: CreateNotification :one
INSERT INTO notification.notification (
  user_id, type, channel, title, body
) VALUES (
  $1, $2, $3, $4, $5
)
RETURNING *;

-- name: ListUserNotifications :many
SELECT * FROM notification.notification
WHERE user_id = $1
ORDER BY created_at DESC;

-- name: MarkNotificationRead :exec
UPDATE notification.notification
SET is_read = true
WHERE id = $1;

-- name: GetUserPreferences :many
SELECT * FROM notification.preference
WHERE user_id = $1;

-- name: UpsertPreference :one
INSERT INTO notification.preference (user_id, channel, enabled)
VALUES ($1, $2, $3)
ON CONFLICT (user_id, channel) DO UPDATE SET enabled = EXCLUDED.enabled
RETURNING *;
