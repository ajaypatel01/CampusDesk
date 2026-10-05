-- A subject's marks distribution for one exam (e.g. Unit Test 1: Written /20;
-- Half Yearly: Written /60 + Oral /10). A row overrides the subject's own
-- fields for that exam only; an empty list means one plain mark. No row means
-- the subject's own fields apply, exactly as before.
CREATE TABLE exam_subject_mark_formats (
    exam_id     UUID NOT NULL REFERENCES exams(id) ON DELETE CASCADE,
    subject_id  UUID NOT NULL REFERENCES subjects(id) ON DELETE CASCADE,
    components  JSONB NOT NULL DEFAULT '[]',
    created_by  UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (exam_id, subject_id)
);
