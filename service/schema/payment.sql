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

CREATE SCHEMA payment;
CREATE TABLE payment.payment (
  id serial PRIMARY KEY,
  user_id int NOT NULL REFERENCES "user"."user" (id),
  gym_id int,
  amount int NOT NULL,
  currency varchar NOT NULL DEFAULT 'usd',
  status varchar NOT NULL DEFAULT 'pending',
  provider varchar DEFAULT 'stripe',
  provider_ref varchar,
  payment_type varchar NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now()
);
