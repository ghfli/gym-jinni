CREATE SCHEMA IF NOT EXISTS booking;

CREATE TABLE IF NOT EXISTS booking.booking (
  id serial PRIMARY KEY,
  user_id int NOT NULL REFERENCES "user"."user" (id),
  class_id int NOT NULL REFERENCES "class"."class" (id),
  status varchar NOT NULL DEFAULT 'confirmed',
  booked_at timestamptz NOT NULL DEFAULT now(),
  cancelled_at timestamptz
);

CREATE INDEX IF NOT EXISTS idx_booking_user_id ON booking.booking (user_id);
CREATE INDEX IF NOT EXISTS idx_booking_class_id ON booking.booking (class_id);
