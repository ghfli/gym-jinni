CREATE UNIQUE INDEX booking_confirmed_user_class ON booking.booking(user_id,class_id) WHERE status='confirmed';
