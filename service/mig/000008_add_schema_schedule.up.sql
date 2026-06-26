CREATE SCHEMA IF NOT EXISTS schedule;

CREATE TABLE IF NOT EXISTS schedule.schedule (
  id serial PRIMARY KEY,
  gym_id int,
  trainer_id int,
  class_id int REFERENCES "class"."class" (id),
  recurrence_rule varchar,
  start_date timestamptz NOT NULL,
  end_date timestamptz,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_schedule_gym_id ON schedule.schedule (gym_id);
CREATE INDEX IF NOT EXISTS idx_schedule_class_id ON schedule.schedule (class_id);
CREATE INDEX IF NOT EXISTS idx_schedule_trainer_id ON schedule.schedule (trainer_id);
