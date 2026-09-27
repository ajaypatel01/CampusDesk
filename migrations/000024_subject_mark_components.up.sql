-- Per-subject, teacher-extensible graded components ("sections" -- Oral,
-- Unit Test, Activity, Practical, Written, or any new one a teacher adds).
-- Nothing is seeded here: a subject with no rows keeps behaving exactly as
-- it does today (the fixed kg/primary/middle report-card template scheme,
-- or plain single-number entry for a grade with no template). This table
-- only starts mattering the moment someone adds a component to a specific
-- subject, at which point it becomes authoritative for that subject.
CREATE TABLE subject_mark_components (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    subject_id     UUID NOT NULL REFERENCES subjects(id) ON DELETE CASCADE,
    key            TEXT NOT NULL,
    label          TEXT NOT NULL,
    max_marks      INT NOT NULL CHECK (max_marks > 0),
    sort_order     INT NOT NULL DEFAULT 0,
    created_by     UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (subject_id, key)
);
