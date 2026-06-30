-- Transfer Certificate Records
CREATE TABLE IF NOT EXISTS tc_records (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    school_id         UUID NOT NULL REFERENCES schools(id) ON DELETE CASCADE,
    scholar_number    VARCHAR(50),
    student_name      VARCHAR(200) NOT NULL,
    father_name       VARCHAR(200),
    mother_name       VARCHAR(200),
    dob               DATE,
    caste             VARCHAR(100),
    category          VARCHAR(100),
    date_of_admission DATE,
    application_date  DATE,
    issue_date        DATE,
    class_passed      VARCHAR(50),
    pen_number        VARCHAR(50),
    apar_id           VARCHAR(50),
    samagra_id        VARCHAR(50),
    new_school        VARCHAR(300),
    dice_code         VARCHAR(50),
    remark            VARCHAR(300),
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_tc_records_school ON tc_records(school_id);
CREATE INDEX idx_tc_records_issue_date ON tc_records(school_id, issue_date);

-- Expense Vouchers
CREATE TABLE IF NOT EXISTS vouchers (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    school_id        UUID NOT NULL REFERENCES schools(id) ON DELETE CASCADE,
    date             DATE NOT NULL,
    account_name     VARCHAR(200) NOT NULL,
    payee            VARCHAR(200),
    amount           NUMERIC(12,2) NOT NULL DEFAULT 0,
    description      TEXT,
    mode_of_payment  VARCHAR(50),
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_vouchers_school ON vouchers(school_id);
CREATE INDEX idx_vouchers_date ON vouchers(school_id, date);
