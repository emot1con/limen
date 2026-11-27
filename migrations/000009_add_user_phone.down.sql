ALTER TABLE "users" DROP CONSTRAINT IF EXISTS "users_phone_format_check";
ALTER TABLE "users" DROP COLUMN IF EXISTS "phone";
