-- Put back the dates 000035 changed, only where the payment still has the
-- corrected date (a later manual edit is kept).
UPDATE fee_payments fp
SET payment_date = f.old_date
FROM fee_payment_date_fix_000035 f
WHERE fp.id = f.payment_id AND fp.payment_date = f.new_date;

DROP TABLE IF EXISTS fee_payment_date_fix_000035;
