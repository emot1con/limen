ALTER TABLE "schedules"
  ADD CONSTRAINT "schedules_day_of_week_check" CHECK ("day_of_week" BETWEEN 1 AND 7),
  ADD CONSTRAINT "schedules_time_range_check" CHECK ("start_time" < "end_time");
