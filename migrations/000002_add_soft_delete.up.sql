ALTER TABLE "users" ADD COLUMN "deleted_at" timestamptz;
ALTER TABLE "students" ADD COLUMN "deleted_at" timestamptz;
ALTER TABLE "teachers" ADD COLUMN "deleted_at" timestamptz;
ALTER TABLE "classes" ADD COLUMN "deleted_at" timestamptz;
ALTER TABLE "subjects" ADD COLUMN "deleted_at" timestamptz;

ALTER TABLE "assessments" ADD COLUMN "deleted_at" timestamptz;
ALTER TABLE "assessment_scores" ADD COLUMN "deleted_at" timestamptz;

ALTER TABLE "attendance_records" ADD COLUMN "deleted_at" timestamptz;

ALTER TABLE "assignments" ADD COLUMN "deleted_at" timestamptz;
ALTER TABLE "assignment_submissions" ADD COLUMN "deleted_at" timestamptz;

ALTER TABLE "quizzes" ADD COLUMN "deleted_at" timestamptz;
ALTER TABLE "quiz_attempts" ADD COLUMN "deleted_at" timestamptz;
ALTER TABLE "quiz_answers" ADD COLUMN "deleted_at" timestamptz;

ALTER TABLE "counseling_notes" ADD COLUMN "deleted_at" timestamptz;
ALTER TABLE "counseling_followups" ADD COLUMN "deleted_at" timestamptz;
ALTER TABLE "aptitude_results" ADD COLUMN "deleted_at" timestamptz;

ALTER TABLE "notifications" ADD COLUMN "deleted_at" timestamptz;

CREATE INDEX ON "users" ("deleted_at");
CREATE INDEX ON "students" ("deleted_at");
CREATE INDEX ON "teachers" ("deleted_at");
CREATE INDEX ON "classes" ("deleted_at");
CREATE INDEX ON "subjects" ("deleted_at");
CREATE INDEX ON "assessments" ("deleted_at");
CREATE INDEX ON "assessment_scores" ("deleted_at");
CREATE INDEX ON "attendance_records" ("deleted_at");
CREATE INDEX ON "assignments" ("deleted_at");
CREATE INDEX ON "assignment_submissions" ("deleted_at");
CREATE INDEX ON "quizzes" ("deleted_at");
CREATE INDEX ON "quiz_attempts" ("deleted_at");
CREATE INDEX ON "quiz_answers" ("deleted_at");
CREATE INDEX ON "counseling_notes" ("deleted_at");
CREATE INDEX ON "counseling_followups" ("deleted_at");
CREATE INDEX ON "aptitude_results" ("deleted_at");
CREATE INDEX ON "notifications" ("deleted_at");
