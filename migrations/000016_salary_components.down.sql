ALTER TABLE staff_profiles
  DROP COLUMN IF EXISTS basic_salary,
  DROP COLUMN IF EXISTS hra,
  DROP COLUMN IF EXISTS special_allowance,
  DROP COLUMN IF EXISTS bonus,
  DROP COLUMN IF EXISTS epf,
  DROP COLUMN IF EXISTS esic,
  DROP COLUMN IF EXISTS additional_deduction,
  DROP COLUMN IF EXISTS additional_deduction_label;
