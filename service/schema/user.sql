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

CREATE TABLE "user"."session" (
  id uuid PRIMARY KEY,
  user_id int REFERENCES "user"."user" (id),
  refresh_tkn_hash varchar NOT NULL,
  user_agent varchar NOT NULL,
  client_ip inet NOT NULL,
  blocked bool NOT NULL DEFAULT false,
  expires_at timestamptz NOT NULL,
  created_at timestamptz NOT NULL
);

CREATE TABLE "user".trainer_profile (
  id serial PRIMARY KEY,
  user_id int NOT NULL UNIQUE REFERENCES "user"."user" (id),
  bio text,
  specializations text[],
  certifications text[],
  hourly_rate int,
  created_at timestamptz NOT NULL DEFAULT now()
);

ALTER TABLE "user".session ADD CONSTRAINT session_refresh_hash CHECK (refresh_tkn_hash ~ '^[0-9a-f]{64}$');
