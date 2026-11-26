DROP INDEX IF EXISTS "questions_deleted_at_idx";
DROP INDEX IF EXISTS "question_banks_deleted_at_idx";
DROP INDEX IF EXISTS "devices_deleted_at_idx";
DROP INDEX IF EXISTS "teaching_assignments_deleted_at_idx";
DROP INDEX IF EXISTS "schedules_deleted_at_idx";
DROP INDEX IF EXISTS "users_email_active_key";

ALTER TABLE "users" ADD CONSTRAINT "users_email_key" UNIQUE ("email");
ALTER TABLE "questions" DROP COLUMN IF EXISTS "deleted_at";
ALTER TABLE "question_banks" DROP COLUMN IF EXISTS "deleted_at";
ALTER TABLE "devices" DROP COLUMN IF EXISTS "deleted_at";
ALTER TABLE "teaching_assignments" DROP COLUMN IF EXISTS "deleted_at";
ALTER TABLE "schedules" DROP COLUMN IF EXISTS "deleted_at";
