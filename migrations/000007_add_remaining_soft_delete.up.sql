ALTER TABLE "users" DROP CONSTRAINT "users_email_key";
CREATE UNIQUE INDEX "users_email_active_key" ON "users" ("email") WHERE "deleted_at" IS NULL;

ALTER TABLE "schedules" ADD COLUMN "deleted_at" timestamptz;
ALTER TABLE "teaching_assignments" ADD COLUMN "deleted_at" timestamptz;
ALTER TABLE "devices" ADD COLUMN "deleted_at" timestamptz;
ALTER TABLE "question_banks" ADD COLUMN "deleted_at" timestamptz;
ALTER TABLE "questions" ADD COLUMN "deleted_at" timestamptz;

CREATE INDEX ON "schedules" ("deleted_at");
CREATE INDEX ON "teaching_assignments" ("deleted_at");
CREATE INDEX ON "devices" ("deleted_at");
CREATE INDEX ON "question_banks" ("deleted_at");
CREATE INDEX ON "questions" ("deleted_at");
