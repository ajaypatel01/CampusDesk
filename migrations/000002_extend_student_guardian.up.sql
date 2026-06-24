ALTER TABLE students
    ADD COLUMN admission_date DATE,
    ADD COLUMN caste VARCHAR(100),
    ADD COLUMN category VARCHAR(50),
    ADD COLUMN aadhar_number VARCHAR(12),
    ADD COLUMN samagra_id VARCHAR(20),
    ADD COLUMN pen_number VARCHAR(30),
    ADD COLUMN apar_id VARCHAR(30),
    ADD COLUMN previous_school VARCHAR(255),
    ADD COLUMN bank_name VARCHAR(150),
    ADD COLUMN bank_ifsc VARCHAR(11),
    ADD COLUMN bank_account_number VARCHAR(30),
    ADD COLUMN bank_holder_name VARCHAR(150),
    ADD COLUMN bank_branch VARCHAR(150);

ALTER TABLE guardians
    ADD COLUMN aadhar_number VARCHAR(12);
