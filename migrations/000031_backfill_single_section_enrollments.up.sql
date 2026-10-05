-- Fill in the class section for current-year enrollments that have none, when
-- the student's grade (from their fee account) has exactly one section that
-- year. Results, homework and broadcasts only reach students through
-- enrollments.class_section_id, so with it NULL class teachers couldn't see
-- or save marks for their own students. Grades with several sections are
-- left alone -- an admin has to pick the section for those.
--
-- The rows changed are recorded so the down migration can undo exactly them.

CREATE TABLE enrollment_section_backfill_000031 (
    enrollment_id    UUID PRIMARY KEY REFERENCES enrollments(id) ON DELETE CASCADE,
    class_section_id UUID NOT NULL
);

WITH single_section AS (
    SELECT academic_year_id, grade_level_id, MIN(id::text)::uuid AS section_id
    FROM class_sections
    GROUP BY academic_year_id, grade_level_id
    HAVING COUNT(*) = 1
),
updated AS (
    UPDATE enrollments e
    SET class_section_id = ss.section_id, updated_at = NOW()
    FROM student_fee_accounts sfa
    JOIN academic_years ay ON ay.id = sfa.academic_year_id AND ay.is_current
    JOIN fee_structures fs ON fs.id = sfa.fee_structure_id
    JOIN single_section ss ON ss.academic_year_id = sfa.academic_year_id AND ss.grade_level_id = fs.grade_level_id
    WHERE e.student_id = sfa.student_id
      AND e.academic_year_id = sfa.academic_year_id
      AND e.class_section_id IS NULL
    RETURNING e.id, e.class_section_id
)
INSERT INTO enrollment_section_backfill_000031 (enrollment_id, class_section_id)
SELECT id, class_section_id FROM updated;
