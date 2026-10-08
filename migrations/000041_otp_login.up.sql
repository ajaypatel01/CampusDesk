-- One-time codes for WhatsApp OTP login and phone verification. Only a
-- hash of the code is stored; a code is good for one use, a few minutes
-- and a few attempts (see internal/modules/user/otp.go).
CREATE TABLE otp_codes (
    id          BIGSERIAL PRIMARY KEY,
    phone       VARCHAR(10) NOT NULL,  -- last 10 digits, e.g. 9876543210
    purpose     TEXT NOT NULL CHECK (purpose IN ('login', 'verify_phone', 'password_reset')),
    code_hash   TEXT NOT NULL,
    attempts    INT NOT NULL DEFAULT 0,
    expires_at  TIMESTAMPTZ NOT NULL,
    used_at     TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX otp_codes_phone_idx ON otp_codes (phone, purpose, created_at DESC);

-- A parent who logs in by phone gets one login linked to every guardian
-- record with that phone (often one per child), so a login may now carry
-- several guardian rows.
DROP INDEX idx_guardians_user_id;
CREATE INDEX idx_guardians_user_id ON guardians(user_id) WHERE user_id IS NOT NULL;
