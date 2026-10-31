ALTER TABLE "class"."class" ADD CONSTRAINT class_time_range CHECK(start_time<end_time);
ALTER TABLE "class"."class" ADD CONSTRAINT class_capacity CHECK(coalesce(min_hdcnt,0)>=0 AND coalesce(max_hdcnt,0)>=0 AND (coalesce(max_hdcnt,0)=0 OR coalesce(min_hdcnt,0)<=max_hdcnt));
