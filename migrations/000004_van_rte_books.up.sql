-- Van management
CREATE TABLE vans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    school_id UUID NOT NULL REFERENCES schools(id) ON DELETE CASCADE,
    van_number VARCHAR(50) NOT NULL,
    driver_name VARCHAR(150) NOT NULL,
    driver_phone VARCHAR(20),
    capacity INT NOT NULL DEFAULT 20,
    route_name VARCHAR(200),
    notes TEXT,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (school_id, van_number)
);

CREATE TABLE van_routes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    van_id UUID NOT NULL REFERENCES vans(id) ON DELETE CASCADE,
    stop_name VARCHAR(150) NOT NULL,
    stop_order INT NOT NULL DEFAULT 0,
    monthly_fee INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE student_van_assignments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id UUID NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    van_id UUID NOT NULL REFERENCES vans(id) ON DELETE CASCADE,
    academic_year_id UUID NOT NULL REFERENCES academic_years(id) ON DELETE CASCADE,
    pickup_stop VARCHAR(150),
    assigned_date DATE NOT NULL DEFAULT CURRENT_DATE,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (student_id, academic_year_id)
);

CREATE INDEX idx_van_assignments_van ON student_van_assignments(van_id, academic_year_id);

-- RTE (Right to Education) quota tracking
CREATE TABLE rte_quotas (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    school_id UUID NOT NULL REFERENCES schools(id) ON DELETE CASCADE,
    academic_year_id UUID NOT NULL REFERENCES academic_years(id) ON DELETE CASCADE,
    grade_level_id UUID NOT NULL REFERENCES grade_levels(id) ON DELETE CASCADE,
    total_seats INT NOT NULL DEFAULT 0,
    govt_reimbursement_per_student INT NOT NULL DEFAULT 0,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (school_id, academic_year_id, grade_level_id)
);

-- Book catalog and lists
CREATE TABLE books (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    school_id UUID NOT NULL REFERENCES schools(id) ON DELETE CASCADE,
    title VARCHAR(300) NOT NULL,
    author VARCHAR(200),
    publisher VARCHAR(200),
    isbn VARCHAR(20),
    price INT NOT NULL DEFAULT 0,
    subject VARCHAR(100),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE book_lists (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    school_id UUID NOT NULL REFERENCES schools(id) ON DELETE CASCADE,
    academic_year_id UUID NOT NULL REFERENCES academic_years(id) ON DELETE CASCADE,
    grade_level_id UUID NOT NULL REFERENCES grade_levels(id) ON DELETE CASCADE,
    name VARCHAR(200) NOT NULL DEFAULT 'Book List',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (school_id, academic_year_id, grade_level_id)
);

CREATE TABLE book_list_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    book_list_id UUID NOT NULL REFERENCES book_lists(id) ON DELETE CASCADE,
    book_id UUID NOT NULL REFERENCES books(id) ON DELETE CASCADE,
    quantity INT NOT NULL DEFAULT 1,
    is_mandatory BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (book_list_id, book_id)
);

CREATE TABLE student_book_receipts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id UUID NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    book_list_id UUID NOT NULL REFERENCES book_lists(id) ON DELETE CASCADE,
    received_date DATE NOT NULL DEFAULT CURRENT_DATE,
    received_by VARCHAR(150),
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (student_id, book_list_id)
);

CREATE INDEX idx_books_school ON books(school_id);
CREATE INDEX idx_book_lists_school_year ON book_lists(school_id, academic_year_id);
CREATE INDEX idx_student_book_receipts_list ON student_book_receipts(book_list_id);
