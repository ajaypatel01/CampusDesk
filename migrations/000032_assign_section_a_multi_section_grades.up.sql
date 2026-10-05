-- Temporary placement: current-year students still without a section go into
-- their grade's section "A" until an admin moves them to the right one. After
-- 000031 these are the students in grades with several sections. Without a
-- section their class teacher can't see them or save their marks.
--
-- The rows changed are recorded so the down migration can undo exactly them.

CREATE TABLE enrollment_section_backfill_000032 (
    enrollment_id    UUID PRIMARY KEY REFERENCES enrollments(id) ON DELETE CASCADE,
    class_section_id UUID NOT NULL
);

WITH updated AS (
    UPDATE enrollments e
    SET class_section_id = cs.id, updated_at = NOW()
    FROM student_fee_accounts sfa
    JOIN academic_years ay ON ay.id = sfa.academic_year_id AND ay.is_current
    JOIN fee_structures fs ON fs.id = sfa.fee_structure_id
    JOIN class_sections cs ON cs.academic_year_id = sfa.academic_year_id
                          AND cs.grade_level_id = fs.grade_level_id
                          AND cs.name = 'A'
    WHERE e.student_id = sfa.student_id
      AND e.academic_year_id = sfa.academic_year_id
      AND e.class_section_id IS NULL
    RETURNING e.id, e.class_section_id
)
INSERT INTO enrollment_section_backfill_000032 (enrollment_id, class_section_id)
SELECT id, class_section_id FROM updated;
