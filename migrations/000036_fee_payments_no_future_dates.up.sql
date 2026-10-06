-- Safeguard: no fee payment may be dated after today (India time), however it
-- is written -- the app already refuses this, but a script or bulk import that
-- writes to the table directly does not go through the app. A day/month mix-up
-- (12 Sep saved as 9 Dec) put 49 payments in the future that way (fixed by
-- 000035). Existing rows are only checked when their date changes, so payments
-- already in the table can still be voided or corrected.
CREATE FUNCTION fee_payments_no_future_date() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    IF (TG_OP = 'INSERT' OR NEW.payment_date IS DISTINCT FROM OLD.payment_date)
       AND NEW.payment_date > (NOW() AT TIME ZONE 'Asia/Kolkata')::date THEN
        RAISE EXCEPTION 'payment date % is in the future -- check the day and month', to_char(NEW.payment_date, 'DD Mon YYYY')
            USING ERRCODE = 'check_violation';
    END IF;
    RETURN NEW;
END;
$$;

CREATE TRIGGER fee_payments_no_future_date
    BEFORE INSERT OR UPDATE OF payment_date ON fee_payments
    FOR EACH ROW EXECUTE FUNCTION fee_payments_no_future_date();
