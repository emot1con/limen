DROP INDEX IF EXISTS "idx_quizzes_exam_type";
ALTER TABLE "quizzes" DROP CONSTRAINT IF EXISTS "quizzes_exam_type_check";
ALTER TABLE "quizzes" DROP COLUMN IF EXISTS "exam_type";
