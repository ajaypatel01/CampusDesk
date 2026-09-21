ALTER TABLE students
  DROP COLUMN IF EXISTS enrollment_number,
  DROP COLUMN IF EXISTS admission_class,
  DROP COLUMN IF EXISTS admission_year;
