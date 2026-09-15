ALTER TABLE staff_leaves DROP CONSTRAINT IF EXISTS staff_leaves_half_day_single_day;
ALTER TABLE staff_leaves DROP COLUMN IF EXISTS half_day;
