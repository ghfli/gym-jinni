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

CREATE SCHEMA gym;
CREATE TABLE gym.gym (
  id serial PRIMARY KEY,
  name varchar NOT NULL,
  address varchar,
  phone varchar,
  email varchar,
  owner_id int REFERENCES "user"."user" (id),
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE gym.membership (
  id serial PRIMARY KEY,
  user_id int NOT NULL REFERENCES "user"."user" (id),
  gym_id int NOT NULL REFERENCES gym.gym (id),
  role varchar NOT NULL DEFAULT 'member',
  joined_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE(user_id, gym_id)
);
