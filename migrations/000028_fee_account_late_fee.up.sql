-- late_fee is a separate, additive charge on a fee account -- it adds to
-- what's owed (total_due/balance) exactly like previous_year_dues does, but
-- is tracked in its own column rather than folded into tuition_fee, so the
-- school's actual tuition amount for a grade is never obscured by a penalty
-- charge layered on top of it.
ALTER TABLE student_fee_accounts ADD COLUMN late_fee INTEGER NOT NULL DEFAULT 0;
