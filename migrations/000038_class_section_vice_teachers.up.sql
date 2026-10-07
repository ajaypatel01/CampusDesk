-- Vice class teachers: any number per section, alongside the one class
-- teacher (class_sections.homeroom_teacher_id). They get the same access to
-- the section's students (results, marks entry) as the class teacher.
CREATE TABLE class_section_vice_teachers (
    class_section_id UUID NOT NULL REFERENCES class_sections(id) ON DELETE CASCADE,
    user_id          UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (class_section_id, user_id)
);
CREATE INDEX idx_class_section_vice_teachers_user ON class_section_vice_teachers(user_id);
