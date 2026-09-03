-- Self-service registration: new users start as "pending" until an admin approves them.
-- Existing rows default to 'approved' so current logins keep working.
ALTER TABLE users ADD COLUMN status VARCHAR(20) NOT NULL DEFAULT 'approved';
ALTER TABLE users ADD CONSTRAINT users_status_check CHECK (status IN ('pending', 'approved', 'rejected'));
CREATE INDEX idx_users_status ON users(status);
