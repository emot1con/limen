DROP INDEX IF EXISTS "users_deleted_at_idx";
DROP INDEX IF EXISTS "students_deleted_at_idx";
DROP INDEX IF EXISTS "teachers_deleted_at_idx";
DROP INDEX IF EXISTS "classes_deleted_at_idx";
DROP INDEX IF EXISTS "subjects_deleted_at_idx";
DROP INDEX IF EXISTS "assessments_deleted_at_idx";
DROP INDEX IF EXISTS "assessment_scores_deleted_at_idx";
DROP INDEX IF EXISTS "attendance_records_deleted_at_idx";
DROP INDEX IF EXISTS "assignments_deleted_at_idx";
DROP INDEX IF EXISTS "assignment_submissions_deleted_at_idx";
DROP INDEX IF EXISTS "quizzes_deleted_at_idx";
DROP INDEX IF EXISTS "quiz_attempts_deleted_at_idx";
DROP INDEX IF EXISTS "quiz_answers_deleted_at_idx";
DROP INDEX IF EXISTS "counseling_notes_deleted_at_idx";
DROP INDEX IF EXISTS "counseling_followups_deleted_at_idx";
DROP INDEX IF EXISTS "aptitude_results_deleted_at_idx";
DROP INDEX IF EXISTS "notifications_deleted_at_idx";

ALTER TABLE "users" DROP COLUMN IF EXISTS "deleted_at";
ALTER TABLE "students" DROP COLUMN IF EXISTS "deleted_at";
ALTER TABLE "teachers" DROP COLUMN IF EXISTS "deleted_at";
ALTER TABLE "classes" DROP COLUMN IF EXISTS "deleted_at";
ALTER TABLE "subjects" DROP COLUMN IF EXISTS "deleted_at";

ALTER TABLE "assessments" DROP COLUMN IF EXISTS "deleted_at";
ALTER TABLE "assessment_scores" DROP COLUMN IF EXISTS "deleted_at";

ALTER TABLE "attendance_records" DROP COLUMN IF EXISTS "deleted_at";

ALTER TABLE "assignments" DROP COLUMN IF EXISTS "deleted_at";
ALTER TABLE "assignment_submissions" DROP COLUMN IF EXISTS "deleted_at";

ALTER TABLE "quizzes" DROP COLUMN IF EXISTS "deleted_at";
ALTER TABLE "quiz_attempts" DROP COLUMN IF EXISTS "deleted_at";
ALTER TABLE "quiz_answers" DROP COLUMN IF EXISTS "deleted_at";

ALTER TABLE "counseling_notes" DROP COLUMN IF EXISTS "deleted_at";
ALTER TABLE "counseling_followups" DROP COLUMN IF EXISTS "deleted_at";
ALTER TABLE "aptitude_results" DROP COLUMN IF EXISTS "deleted_at";

ALTER TABLE "notifications" DROP COLUMN IF EXISTS "deleted_at";
