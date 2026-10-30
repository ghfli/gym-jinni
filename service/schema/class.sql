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

ALTER TABLE "class"."class" ADD CONSTRAINT class_time_range CHECK(start_time<end_time);
ALTER TABLE "class"."class" ADD CONSTRAINT class_capacity CHECK(coalesce(min_hdcnt,0)>=0 AND coalesce(max_hdcnt,0)>=0 AND (coalesce(max_hdcnt,0)=0 OR coalesce(min_hdcnt,0)<=max_hdcnt));
