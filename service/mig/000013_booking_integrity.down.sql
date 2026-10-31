ALTER TABLE "class"."class" DROP CONSTRAINT class_capacity;
ALTER TABLE "class"."class" DROP CONSTRAINT class_time_range;
DROP INDEX booking.booking_confirmed_user_class;
