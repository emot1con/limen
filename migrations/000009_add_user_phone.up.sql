ALTER TABLE "users" ADD COLUMN "phone" text NOT NULL DEFAULT '';
ALTER TABLE "users" ADD CONSTRAINT "users_phone_format_check" CHECK ("phone" = '' OR "phone" ~ '^\\+62[0-9]{8,13}$');
ALTER TABLE "users" ALTER COLUMN "phone" DROP DEFAULT;
