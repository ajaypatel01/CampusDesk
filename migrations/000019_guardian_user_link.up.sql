ALTER TABLE guardians
  ADD COLUMN user_id UUID REFERENCES users(id) ON DELETE SET NULL;

CREATE UNIQUE INDEX idx_guardians_user_id ON guardians(user_id) WHERE user_id IS NOT NULL;
