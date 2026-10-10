-- Homework can carry one image or PDF (stored in S3) besides its text.
ALTER TABLE homework_assignments
    ADD COLUMN attachment_key  TEXT,
    ADD COLUMN attachment_name TEXT,
    ADD COLUMN attachment_type TEXT,
    ADD COLUMN attachment_size INT;
