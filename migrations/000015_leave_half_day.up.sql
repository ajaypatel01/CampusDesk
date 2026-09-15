ALTER TABLE staff_leaves ADD COLUMN half_day BOOLEAN NOT NULL DEFAULT false;

-- A half day only makes sense for a single-day leave record.
ALTER TABLE staff_leaves ADD CONSTRAINT staff_leaves_half_day_single_day
  CHECK (NOT half_day OR start_date = end_date);
