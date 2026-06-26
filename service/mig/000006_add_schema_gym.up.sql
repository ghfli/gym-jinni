CREATE SCHEMA IF NOT EXISTS gym;

CREATE TABLE IF NOT EXISTS gym.gym (
  id serial PRIMARY KEY,
  name varchar NOT NULL,
  address varchar,
  phone varchar,
  email varchar,
  owner_id int REFERENCES "user"."user" (id),
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS gym.membership (
  id serial PRIMARY KEY,
  user_id int NOT NULL REFERENCES "user"."user" (id),
  gym_id int NOT NULL REFERENCES gym.gym (id),
  role varchar NOT NULL DEFAULT 'member',
  joined_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE(user_id, gym_id)
);

CREATE INDEX IF NOT EXISTS idx_gym_owner_id ON gym.gym (owner_id);
CREATE INDEX IF NOT EXISTS idx_membership_user_id ON gym.membership (user_id);
CREATE INDEX IF NOT EXISTS idx_membership_gym_id ON gym.membership (gym_id);
