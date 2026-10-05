-- Whether a subject with no stored mark components falls back to its grade's
-- report-card template fields (Written/Note Book/Oral/...). Every existing
-- subject keeps that behaviour (DEFAULT true), so nothing already recorded
-- changes meaning; the app creates new subjects with false, so they start
-- with no fields and an admin adds only the ones they want.
ALTER TABLE subjects ADD COLUMN uses_template_components BOOLEAN NOT NULL DEFAULT true;
