DROP INDEX IF EXISTS idx_guardians_user_id;
ALTER TABLE guardians DROP COLUMN IF EXISTS user_id;
