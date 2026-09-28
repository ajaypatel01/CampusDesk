-- phone_number is nullable (most accounts still won't have one) but must be
-- unique once set -- a partial index rather than a plain UNIQUE constraint
-- so it doesn't reject the many NULL rows (NULL <> NULL, but a plain UNIQUE
-- index still only allows one NULL per Postgres's actual behavior; making
-- the intent explicit with a partial index is clearer and matches the
-- sentinel-value pattern used elsewhere in this schema).
ALTER TABLE users ADD COLUMN phone_number VARCHAR(20);
CREATE UNIQUE INDEX idx_users_phone_number ON users(phone_number) WHERE phone_number IS NOT NULL;
