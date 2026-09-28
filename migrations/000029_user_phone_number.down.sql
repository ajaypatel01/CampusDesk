DROP INDEX IF EXISTS idx_users_phone_number;
ALTER TABLE users DROP COLUMN IF EXISTS phone_number;
