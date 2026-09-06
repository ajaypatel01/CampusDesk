ALTER TABLE staff_profiles ADD COLUMN cl_quota_per_year INTEGER NOT NULL DEFAULT 12;
ALTER TABLE schools ADD COLUMN working_days_per_month INTEGER NOT NULL DEFAULT 26;

CREATE TABLE staff_leaves (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id          UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    academic_year_id UUID NOT NULL REFERENCES academic_years(id) ON DELETE CASCADE,
    leave_type       VARCHAR(20) NOT NULL DEFAULT 'cl' CHECK (leave_type IN ('cl', 'unpaid')),
    start_date       DATE NOT NULL,
    end_date         DATE NOT NULL,
    reason           TEXT,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (end_date >= start_date)
);

CREATE INDEX idx_staff_leaves_user ON staff_leaves(user_id);
CREATE INDEX idx_staff_leaves_year ON staff_leaves(academic_year_id);
