ALTER TABLE grade_levels
    ADD COLUMN report_card_template TEXT
        CHECK (report_card_template IN ('kg', 'primary', 'middle'));

-- Backfill by name for the grades whose printed report-card format is already
-- known (Nursery/LKG/UKG, 1st-4th, 6th-7th). 5th and 8th-12th stay NULL and
-- keep using the existing generic single-exam marksheet until a template for
-- them exists.
UPDATE grade_levels SET report_card_template = 'kg' WHERE name IN ('Nursery', 'LKG', 'UKG');
UPDATE grade_levels SET report_card_template = 'primary' WHERE name IN ('1st', '2nd', '3rd', '4th');
UPDATE grade_levels SET report_card_template = 'middle' WHERE name IN ('6th', '7th');

-- IF NOT EXISTS: dice_code was added ahead of this migration to unblock
-- setting the two schools' real DICE codes before this branch was merged.
ALTER TABLE schools
    ADD COLUMN IF NOT EXISTS dice_code TEXT;

-- Component-level breakdown (Written/Note Book/Activity/Oral/Test, or
-- Test/Project/Theory) under an exam_marks row. exam_marks.marks_obtained and
-- .max_marks keep holding the computed total, so the existing single-exam
-- marksheet/PDF and any mark with no components (older data, or a grade with
-- no report_card_template) keep working unchanged.
CREATE TABLE exam_mark_components (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    exam_mark_id   UUID NOT NULL REFERENCES exam_marks(id) ON DELETE CASCADE,
    component_key  TEXT NOT NULL,
    obtained       NUMERIC(6,2) NOT NULL,
    max_marks      INT NOT NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (exam_mark_id, component_key)
);

-- The handwritten-on-paper fields of a report card, one row per student per
-- year (not per exam -- they're filled in once for the combined report).
CREATE TABLE report_card_details (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    school_id         UUID NOT NULL REFERENCES schools(id) ON DELETE CASCADE,
    academic_year_id  UUID NOT NULL REFERENCES academic_years(id) ON DELETE CASCADE,
    grade_level_id    UUID NOT NULL REFERENCES grade_levels(id) ON DELETE CASCADE,
    student_id        UUID NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    roll_no           TEXT,
    attendance        TEXT,
    remark            TEXT,
    promoted_to       TEXT,
    moral_remark      TEXT,
    gk_remark         TEXT,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (student_id, academic_year_id)
);

-- Qualitative Co-Scholastic/Discipline grades (Work Education, Arts, ... --
-- the 'middle' template's 9 fixed criteria). Not tied to a Subject: these are
-- typed grades, never computed from marks.
CREATE TABLE discipline_grades (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    school_id         UUID NOT NULL REFERENCES schools(id) ON DELETE CASCADE,
    academic_year_id  UUID NOT NULL REFERENCES academic_years(id) ON DELETE CASCADE,
    student_id        UUID NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    criterion_key     TEXT NOT NULL,
    grade             TEXT NOT NULL,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (student_id, academic_year_id, criterion_key)
);
