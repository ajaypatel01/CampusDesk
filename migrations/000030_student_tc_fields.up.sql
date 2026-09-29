-- Recorded when a student is marked inactive (leaving the school) -- see
-- Service.Update: both are required whenever status is set to 'inactive',
-- since that transition is what a Transfer Certificate is actually issued
-- for.
ALTER TABLE students ADD COLUMN tc_date DATE;
ALTER TABLE students ADD COLUMN tc_year VARCHAR(20);
