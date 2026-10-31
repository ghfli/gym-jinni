CREATE SCHEMA IF NOT EXISTS payment;

CREATE TABLE IF NOT EXISTS payment.payment (
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

CREATE INDEX IF NOT EXISTS idx_payment_user_id ON payment.payment (user_id);
