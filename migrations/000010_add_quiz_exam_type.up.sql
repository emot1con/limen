ALTER TABLE "quizzes" ADD COLUMN "exam_type" text NOT NULL DEFAULT 'latihan';
ALTER TABLE "quizzes" ADD CONSTRAINT "quizzes_exam_type_check" CHECK ("exam_type" IN ('latihan', 'ulangan_harian', 'pts', 'pas'));
CREATE INDEX "idx_quizzes_exam_type" ON "quizzes" ("exam_type");
