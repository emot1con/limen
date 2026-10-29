ALTER TABLE "schedules"
  DROP CONSTRAINT IF EXISTS "schedules_time_range_check",
  DROP CONSTRAINT IF EXISTS "schedules_day_of_week_check";
