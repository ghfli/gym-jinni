CREATE SCHEMA IF NOT EXISTS notification;

CREATE TABLE IF NOT EXISTS notification.notification (
  id serial PRIMARY KEY,
  user_id int NOT NULL REFERENCES "user"."user" (id),
  type varchar NOT NULL,
  channel varchar NOT NULL DEFAULT 'in_app',
  title varchar NOT NULL,
  body text,
  is_read bool NOT NULL DEFAULT false,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS notification.preference (
  id serial PRIMARY KEY,
  user_id int NOT NULL REFERENCES "user"."user" (id),
  channel varchar NOT NULL,
  enabled bool NOT NULL DEFAULT true,
  UNIQUE(user_id, channel)
);

CREATE INDEX IF NOT EXISTS idx_notification_user_id ON notification.notification (user_id);
CREATE INDEX IF NOT EXISTS idx_preference_user_id ON notification.preference (user_id);
