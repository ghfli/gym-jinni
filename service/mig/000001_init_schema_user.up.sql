-- based on ../gen/sql/user.dbml.sql
-- update me whenever it changes

CREATE SCHEMA If not exists "user";

CREATE TABLE if not exists "user"."user" (
  "id" serial PRIMARY KEY,
  "email" varchar,
  "phone" varchar,
  "name" varchar NOT NULL,
  "hashed_passwd" varchar NOT NULL,
  "email_verified" bool DEFAULT false,
  "phone_verified" bool DEFAULT false,
  "created_at" timestamptz NOT NULL DEFAULT (now()),
  "passwd_changed_at" timestamptz NOT NULL DEFAULT '0001-01-01'
);

CREATE TABLE if not exists "user"."session" (
  "id" uuid PRIMARY KEY,
  "user_id" int,
  "refresh_tkn" varchar NOT NULL,
  "user_agent" varchar NOT NULL,
  "client_ip" inet NOT NULL,
  "blocked" bool NOT NULL DEFAULT false,
  "expires_at" timestamptz NOT NULL,
  "created_at" timestamptz NOT NULL
);

ALTER TABLE "user"."session" ADD FOREIGN KEY ("user_id") REFERENCES "user"."user" ("id");
