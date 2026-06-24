DROP INDEX IF EXISTS idx_fee_payments_date;
DROP INDEX IF EXISTS idx_fee_payments_account;
DROP INDEX IF EXISTS idx_student_fee_accounts_student;
DROP INDEX IF EXISTS idx_student_fee_accounts_school_year;
DROP INDEX IF EXISTS idx_fee_structures_school_year;

DROP TABLE IF EXISTS fee_payments;
DROP TABLE IF EXISTS student_fee_accounts;
DROP TABLE IF EXISTS fee_installment_plans;
DROP TABLE IF EXISTS fee_structures;
