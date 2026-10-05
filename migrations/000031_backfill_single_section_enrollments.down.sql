-- Clear only the sections 000031 filled in, and only where the enrollment
-- still has the section it was given (an admin's later change is kept).
UPDATE enrollments e
SET class_section_id = NULL, updated_at = NOW()
FROM enrollment_section_backfill_000031 b
WHERE e.id = b.enrollment_id
  AND e.class_section_id = b.class_section_id;

DROP TABLE IF EXISTS enrollment_section_backfill_000031;
