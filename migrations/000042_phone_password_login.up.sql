-- Parents log in with a mobile number and a default password (any linked
-- child's first name + "@123") until they set their own; own_password
-- records that they have, after which the default stops working.
ALTER TABLE users ADD COLUMN own_password BOOLEAN NOT NULL DEFAULT FALSE;

-- Failed mobile-number logins, for the "5 wrong tries in 15 minutes" limit.
CREATE TABLE login_failures (
    id     BIGSERIAL PRIMARY KEY,
    phone  VARCHAR(10) NOT NULL,
    at     TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX login_failures_phone_at_idx ON login_failures (phone, at DESC);
