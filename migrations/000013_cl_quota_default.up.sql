ALTER TABLE staff_profiles ALTER COLUMN cl_quota_per_year SET DEFAULT 7;

-- Reset anyone still sitting at the old default of 12 (nobody has customized it yet)
-- to the new standard of 7. Staff-specific exceptions (e.g. staff with no CL
-- entitlement) are handled separately per person, not by this bulk update.
UPDATE staff_profiles SET cl_quota_per_year = 7 WHERE cl_quota_per_year = 12;
