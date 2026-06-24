CREATE TABLE fee_structures (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    school_id UUID NOT NULL REFERENCES schools(id) ON DELETE CASCADE,
    academic_year_id UUID NOT NULL REFERENCES academic_years(id) ON DELETE CASCADE,
    grade_level_id UUID NOT NULL REFERENCES grade_levels(id) ON DELETE CASCADE,
    tuition_fee_annual INTEGER NOT NULL,
    num_installments INTEGER NOT NULL DEFAULT 4,
    van_fee_annual INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (school_id, academic_year_id, grade_level_id)
);

CREATE TABLE fee_installment_plans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    fee_structure_id UUID NOT NULL REFERENCES fee_structures(id) ON DELETE CASCADE,
    installment_number INTEGER NOT NULL,
    label VARCHAR(100) NOT NULL,
    amount INTEGER NOT NULL,
    due_date DATE NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (fee_structure_id, installment_number)
);

CREATE TABLE student_fee_accounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id UUID NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    school_id UUID NOT NULL REFERENCES schools(id) ON DELETE CASCADE,
    academic_year_id UUID NOT NULL REFERENCES academic_years(id) ON DELETE CASCADE,
    fee_structure_id UUID NOT NULL REFERENCES fee_structures(id) ON DELETE RESTRICT,
    tuition_fee INTEGER NOT NULL,
    discount_amount INTEGER NOT NULL DEFAULT 0,
    discount_reason VARCHAR(500),
    previous_year_dues INTEGER NOT NULL DEFAULT 0,
    van_fee INTEGER NOT NULL DEFAULT 0,
    is_rte BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (student_id, academic_year_id)
);

CREATE TABLE fee_payments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_fee_account_id UUID NOT NULL REFERENCES student_fee_accounts(id) ON DELETE CASCADE,
    fee_type VARCHAR(30) NOT NULL,
    installment_number INTEGER,
    amount INTEGER NOT NULL,
    payment_date DATE NOT NULL,
    payment_mode VARCHAR(20) NOT NULL DEFAULT 'cash',
    reference_number VARCHAR(100),
    notes TEXT,
    voided BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_fee_structures_school_year ON fee_structures(school_id, academic_year_id);
CREATE INDEX idx_student_fee_accounts_school_year ON student_fee_accounts(school_id, academic_year_id);
CREATE INDEX idx_student_fee_accounts_student ON student_fee_accounts(student_id);
CREATE INDEX idx_fee_payments_account ON fee_payments(student_fee_account_id);
CREATE INDEX idx_fee_payments_date ON fee_payments(payment_date);
