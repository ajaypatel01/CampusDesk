-- Results fee lock, set per exam: when on, a parent whose fee balance for
-- the exam's academic year is more than fee_lock_min_due can't see that
-- exam's results (staff still can). 0 means any amount due locks them.
ALTER TABLE exams
    ADD COLUMN fee_lock_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN fee_lock_min_due INT NOT NULL DEFAULT 0 CHECK (fee_lock_min_due >= 0);
