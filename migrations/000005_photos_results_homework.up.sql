-- Student and teacher photos + blood group
ALTER TABLE students ADD COLUMN IF NOT EXISTS photo_url TEXT;
ALTER TABLE students ADD COLUMN IF NOT EXISTS blood_group VARCHAR(10);
ALTER TABLE users ADD COLUMN IF NOT EXISTS photo_url TEXT;
ALTER TABLE users ADD COLUMN IF NOT EXISTS employee_id VARCHAR(50);
ALTER TABLE users ADD COLUMN IF NOT EXISTS department VARCHAR(100);

-- Subjects per grade
CREATE TABLE subjects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    school_id UUID NOT NULL REFERENCES schools(id) ON DELETE CASCADE,
    grade_level_id UUID NOT NULL REFERENCES grade_levels(id) ON DELETE CASCADE,
    name VARCHAR(150) NOT NULL,
    code VARCHAR(20),
    max_marks INT NOT NULL DEFAULT 100,
    passing_marks INT NOT NULL DEFAULT 33,
    sort_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (school_id, grade_level_id, name)
);

-- Exams (Unit Test 1, Half-Yearly, Annual, etc.)
CREATE TABLE exams (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    school_id UUID NOT NULL REFERENCES schools(id) ON DELETE CASCADE,
    academic_year_id UUID NOT NULL REFERENCES academic_years(id) ON DELETE CASCADE,
    grade_level_id UUID NOT NULL REFERENCES grade_levels(id) ON DELETE CASCADE,
    name VARCHAR(150) NOT NULL,
    exam_date DATE,
    weight_percent INT NOT NULL DEFAULT 100,
    is_published BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Per-student per-subject marks
CREATE TABLE exam_marks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    exam_id UUID NOT NULL REFERENCES exams(id) ON DELETE CASCADE,
    student_id UUID NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    subject_id UUID NOT NULL REFERENCES subjects(id) ON DELETE CASCADE,
    marks_obtained NUMERIC(6,2) NOT NULL DEFAULT 0,
    max_marks INT NOT NULL DEFAULT 100,
    is_absent BOOLEAN NOT NULL DEFAULT FALSE,
    remarks TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (exam_id, student_id, subject_id)
);

-- Homework assignments
CREATE TABLE homework_assignments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    school_id UUID NOT NULL REFERENCES schools(id) ON DELETE CASCADE,
    academic_year_id UUID NOT NULL REFERENCES academic_years(id) ON DELETE CASCADE,
    grade_level_id UUID NOT NULL REFERENCES grade_levels(id) ON DELETE CASCADE,
    class_section_id UUID REFERENCES class_sections(id) ON DELETE SET NULL,
    subject_id UUID REFERENCES subjects(id) ON DELETE SET NULL,
    title VARCHAR(300) NOT NULL,
    description TEXT,
    assigned_by UUID REFERENCES users(id) ON DELETE SET NULL,
    assigned_date DATE NOT NULL DEFAULT CURRENT_DATE,
    due_date DATE NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Per-student homework status
CREATE TABLE homework_submissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    assignment_id UUID NOT NULL REFERENCES homework_assignments(id) ON DELETE CASCADE,
    student_id UUID NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    submitted_date DATE,
    status VARCHAR(30) NOT NULL DEFAULT 'pending',
    remarks TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (assignment_id, student_id)
);

CREATE INDEX idx_exam_marks_exam ON exam_marks(exam_id);
CREATE INDEX idx_exam_marks_student ON exam_marks(student_id);
CREATE INDEX idx_homework_grade ON homework_assignments(school_id, academic_year_id, grade_level_id);
CREATE INDEX idx_homework_submissions_assignment ON homework_submissions(assignment_id);
