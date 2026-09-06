-- Normalize existing values, then restrict gender to male/female (or unset) going forward.
UPDATE students SET gender = LOWER(TRIM(gender)) WHERE gender IS NOT NULL;
UPDATE students SET gender = NULL WHERE TRIM(gender) = '';

ALTER TABLE students ADD CONSTRAINT students_gender_check
    CHECK (gender IS NULL OR gender IN ('male', 'female'));
