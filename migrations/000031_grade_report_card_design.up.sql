-- Locks a report card's visual design (classic/modern/minimal) per grade
-- level, same shape as the existing report_card_template column: once an
-- admin sets it under Settings, every student in that grade sees/downloads
-- their report card in that design -- it's no longer a free per-download
-- choice.
ALTER TABLE grade_levels ADD COLUMN report_card_design VARCHAR(20);
