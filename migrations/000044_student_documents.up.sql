-- Scans/photos of a student's papers (bank passbook, Aadhaar card, ...),
-- stored privately in S3; staff only. meta keeps a summary of the details
-- entered with the upload (numbers only as their last 4 digits -- the full
-- values live on the student's own record).
CREATE TABLE student_documents (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    school_id    UUID NOT NULL REFERENCES schools(id) ON DELETE CASCADE,
    student_id   UUID NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    doc_type     TEXT NOT NULL CHECK (doc_type IN (
        'bank_passbook', 'aadhar_card', 'samagra_id', 'caste_certificate', 'birth_certificate',
        'income_certificate', 'domicile_certificate', 'transfer_certificate', 'marksheet', 'photo', 'other')),
    title        TEXT NOT NULL DEFAULT '',
    meta         JSONB NOT NULL DEFAULT '{}',
    file_name    TEXT NOT NULL,
    s3_key       TEXT NOT NULL,
    content_type TEXT NOT NULL,
    size_bytes   INT NOT NULL,
    uploaded_by  UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX student_documents_student_idx ON student_documents (student_id, created_at DESC);

SELECT audit_enable('student_documents');
