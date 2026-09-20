ALTER TABLE staff_profiles
  ADD COLUMN basic_salary INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN hra INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN special_allowance INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN bonus INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN epf INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN esic INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN additional_deduction INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN additional_deduction_label VARCHAR(200);

-- Existing staff only ever had a single flat "salary" figure; treat it as
-- their basic pay so gross salary (basic+hra+special_allowance+bonus) stays
-- unchanged for everyone until an admin breaks it down further.
UPDATE staff_profiles SET basic_salary = COALESCE(salary, 0);
