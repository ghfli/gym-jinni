-- Existing invalid/duplicate data must be reviewed before this migration;
-- do not silently cancel customer bookings or rewrite class times.
CREATE UNIQUE INDEX booking_confirmed_user_class ON booking.booking(user_id,class_id) WHERE status='confirmed';
ALTER TABLE "class"."class" ADD CONSTRAINT class_time_range CHECK(start_time<end_time);
ALTER TABLE "class"."class" ADD CONSTRAINT class_capacity CHECK(coalesce(min_hdcnt,0)>=0 AND coalesce(max_hdcnt,0)>=0 AND (coalesce(max_hdcnt,0)=0 OR coalesce(min_hdcnt,0)<=max_hdcnt));
