CREATE SCHEMA "user";
CREATE TABLE "user"."user" (
  id serial PRIMARY KEY,
  email varchar,
  phone varchar,
  name varchar NOT NULL,
  hashed_passwd varchar NOT NULL,
  email_verified bool DEFAULT false,
  phone_verified bool DEFAULT false,
  created_at timestamptz NOT NULL DEFAULT (now()),
  passwd_changed_at timestamptz NOT NULL DEFAULT '0001-01-01'
);

CREATE SCHEMA "class";
CREATE TABLE "class"."class" (
  id serial PRIMARY KEY,
  created_by int REFERENCES "user"."user" (id),
  description varchar NOT NULL,
  start_time timestamptz NOT NULL,
  end_time timestamptz NOT NULL,
  min_hdcnt int,
  max_hdcnt int
);

CREATE SCHEMA booking;
CREATE TABLE booking.booking (
  id serial PRIMARY KEY,
  user_id int NOT NULL REFERENCES "user"."user" (id),
  class_id int NOT NULL REFERENCES "class"."class" (id),
  status varchar NOT NULL DEFAULT 'confirmed',
  booked_at timestamptz NOT NULL DEFAULT now(),
  cancelled_at timestamptz
);
