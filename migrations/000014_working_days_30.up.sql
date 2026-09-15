ALTER TABLE schools ALTER COLUMN working_days_per_month SET DEFAULT 30;

-- Salary is calculated out of a 30-day month at both schools, not a 26-day
-- "working days" basis — reset anyone still on the old default.
UPDATE schools SET working_days_per_month = 30 WHERE working_days_per_month = 26;
