ALTER TABLE staff_profiles
    ADD COLUMN IF NOT EXISTS staff_type VARCHAR(20) DEFAULT 'teaching';

UPDATE staff_profiles
SET staff_type = 'non_teaching'
WHERE designation IN ('Driver', 'School Maid', 'ECO', 'Receptionist', 'Peon', 'Guard', 'Cook', 'Accountant', 'Clerk');
