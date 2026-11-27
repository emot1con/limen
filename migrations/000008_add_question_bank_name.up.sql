ALTER TABLE "question_banks" ADD COLUMN "name" text NOT NULL DEFAULT '';
ALTER TABLE "question_banks" ALTER COLUMN "name" DROP DEFAULT;
