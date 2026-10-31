CREATE TABLE IF NOT EXISTS "user".trainer_profile (
  id serial PRIMARY KEY,
  user_id int NOT NULL UNIQUE REFERENCES "user"."user" (id),
  bio text,
  specializations text[],
  certifications text[],
  hourly_rate int,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_trainer_profile_user_id ON "user".trainer_profile (user_id);
