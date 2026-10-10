-- Class gallery and class documents: files teachers (and admins) upload
-- for a class section, stored in S3; parents see their child's section.
CREATE TABLE class_media (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    school_id        UUID NOT NULL REFERENCES schools(id) ON DELETE CASCADE,
    class_section_id UUID NOT NULL REFERENCES class_sections(id) ON DELETE CASCADE,
    kind             TEXT NOT NULL CHECK (kind IN ('photo', 'document')),
    event            TEXT NOT NULL DEFAULT '',   -- e.g. "Annual Day"; groups photos into albums
    event_date       DATE,
    file_name        TEXT NOT NULL,
    s3_key           TEXT NOT NULL,
    thumb_key        TEXT,                       -- small JPEG for photos, when one could be made
    content_type     TEXT NOT NULL,
    size_bytes       INT NOT NULL,
    uploaded_by      UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX class_media_section_idx ON class_media (class_section_id, kind, created_at DESC);

-- A copy of every TC, marksheet and report card PDF that is issued, kept in
-- S3 exactly as it was issued. Identical copies (same sha256) are stored once.
CREATE TABLE issued_documents (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    school_id    UUID NOT NULL REFERENCES schools(id) ON DELETE CASCADE,
    student_id   UUID REFERENCES students(id) ON DELETE SET NULL,
    student_name TEXT NOT NULL,
    kind         TEXT NOT NULL CHECK (kind IN ('tc', 'marksheet', 'report_card')),
    title        TEXT NOT NULL,
    s3_key       TEXT NOT NULL,
    sha256       TEXT NOT NULL,
    size_bytes   INT NOT NULL,
    issued_by    UUID REFERENCES users(id) ON DELETE SET NULL,
    issued_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (student_id, kind, sha256)
);
CREATE INDEX issued_documents_student_idx ON issued_documents (student_id, issued_at DESC);

SELECT audit_enable('class_media');
SELECT audit_enable('issued_documents');
