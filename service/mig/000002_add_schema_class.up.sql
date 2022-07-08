-- based on ../gen/sql/class.dbml.sql
-- Update me whenever it changes

CREATE SCHEMA if not exists  "class";

CREATE TABLE  if not exists "class"."class" (
  "id" serial PRIMARY KEY,
  "created_by" int,
  "description" varchar NOT NULL,
  "start_time" timestamptz NOT NULL,
  "end_time" timestamptz NOT NULL,
  "min_hdcnt" int,
  "max_hdcnt" int
);

ALTER TABLE "class"."class" ADD FOREIGN KEY ("created_by") REFERENCES "user"."user" ("id");
