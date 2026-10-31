CREATE SCHEMA IF NOT EXISTS activity;

CREATE TABLE IF NOT EXISTS activity.activity (
  id serial PRIMARY KEY,
  user_id int NOT NULL REFERENCES "user"."user" (id),
  gym_id int,
  activity_type varchar NOT NULL,
  title varchar NOT NULL,
  description text,
  duration_minutes int,
  calories int,
  distance_meters int,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_activity_user_id ON activity.activity (user_id);
