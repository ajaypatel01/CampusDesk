ALTER TABLE guardians DROP COLUMN IF EXISTS aadhar_number;

ALTER TABLE students
    DROP COLUMN IF EXISTS admission_date,
    DROP COLUMN IF EXISTS caste,
    DROP COLUMN IF EXISTS category,
    DROP COLUMN IF EXISTS aadhar_number,
    DROP COLUMN IF EXISTS samagra_id,
    DROP COLUMN IF EXISTS pen_number,
    DROP COLUMN IF EXISTS apar_id,
    DROP COLUMN IF EXISTS previous_school,
    DROP COLUMN IF EXISTS bank_name,
    DROP COLUMN IF EXISTS bank_ifsc,
    DROP COLUMN IF EXISTS bank_account_number,
    DROP COLUMN IF EXISTS bank_holder_name,
    DROP COLUMN IF EXISTS bank_branch;
