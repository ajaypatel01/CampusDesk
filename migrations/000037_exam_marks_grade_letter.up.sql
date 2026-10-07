-- Grading-only (co-scholastic) subjects are marked with a letter grade, not
-- numbers. NULL for every ordinary mark and for co-scholastic marks entered
-- as numbers before this existed.
ALTER TABLE exam_marks ADD COLUMN grade_letter VARCHAR(2)
    CHECK (grade_letter IN ('A', 'B', 'C', 'D'));
