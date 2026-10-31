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

CREATE SCHEMA notification;
CREATE TABLE notification.notification (
  id serial PRIMARY KEY,
  user_id int NOT NULL REFERENCES "user"."user" (id),
  type varchar NOT NULL,
  channel varchar NOT NULL DEFAULT 'in_app',
  title varchar NOT NULL,
  body text,
  is_read bool NOT NULL DEFAULT false,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE notification.preference (
  id serial PRIMARY KEY,
  user_id int NOT NULL REFERENCES "user"."user" (id),
  channel varchar NOT NULL,
  enabled bool NOT NULL DEFAULT true,
  UNIQUE(user_id, channel)
);
