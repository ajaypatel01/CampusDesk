CREATE TABLE IF NOT EXISTS staff_profiles (
    id                         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id                    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    guardian_name              VARCHAR(200),
    aadhar_number              VARCHAR(20),
    education_qualification    VARCHAR(200),
    professional_qualification VARCHAR(200),
    designation                VARCHAR(100),
    salary                     INTEGER DEFAULT 0,
    bank_name                  VARCHAR(100),
    bank_ifsc                  VARCHAR(20),
    bank_branch                VARCHAR(100),
    bank_account_number        VARCHAR(50),
    bank_account_holder        VARCHAR(200),
    phone                      VARCHAR(20),
    created_at                 TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at                 TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id)
);

CREATE INDEX idx_staff_profiles_user ON staff_profiles(user_id);
