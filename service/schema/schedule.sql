CREATE SCHEMA "class";
CREATE TABLE "class"."class" (
  id serial PRIMARY KEY,
  created_by int,
  description varchar NOT NULL,
  start_time timestamptz NOT NULL,
  end_time timestamptz NOT NULL,
  min_hdcnt int,
  max_hdcnt int
);

CREATE SCHEMA schedule;
CREATE TABLE schedule.schedule (
  id serial PRIMARY KEY,
  gym_id int,
  trainer_id int,
  class_id int REFERENCES "class"."class" (id),
  recurrence_rule varchar,
  start_date timestamptz NOT NULL,
  end_date timestamptz,
  created_at timestamptz NOT NULL DEFAULT now()
);
