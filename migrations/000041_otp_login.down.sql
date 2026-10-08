DROP INDEX idx_guardians_user_id;
CREATE UNIQUE INDEX idx_guardians_user_id ON guardians(user_id) WHERE user_id IS NOT NULL;
DROP TABLE otp_codes;
