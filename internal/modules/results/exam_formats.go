package results

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/ajaypatel01/CampusDesk/internal/platform/database"
	apperr "github.com/ajaypatel01/CampusDesk/internal/platform/errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v4"
)

// A subject's marks distribution can differ per exam (Unit Test: Written /20,
// Half Yearly: Written /60 + Oral /10). exam_subject_mark_formats holds those
// per-exam overrides; a subject with no row for an exam uses its own fields
// (GetSubjectComponents), exactly as before per-exam formats existed.

// ExamSubjectFormat is one subject's effective scheme in one exam.
type ExamSubjectFormat struct {
	SubjectID   uuid.UUID       `json:"subject_id"`
	SubjectName string          `json:"subject_name"`
	MaxMarks    int             `json:"max_marks"`
	Components  []MarkComponent `json:"components"`
	// Custom is true when the exam has its own format for this subject,
	// false when the subject's own fields apply.
	Custom bool `json:"custom"`
}

type examScope struct {
	SchoolID, AcademicYearID, GradeLevelID uuid.UUID
}

func (r *Repository) examScope(ctx context.Context, examID uuid.UUID) (examScope, error) {
	var s examScope
	err := r.pool.QueryRow(ctx, `SELECT school_id, academic_year_id, grade_level_id FROM exams WHERE id=$1`, examID).
		Scan(&s.SchoolID, &s.AcademicYearID, &s.GradeLevelID)
	if errors.Is(err, pgx.ErrNoRows) {
		return s, apperr.ErrNotFound
	}
	return s, err
}

// ExamGradeAndYear returns the grade and academic year an exam belongs to.
func (r *Repository) ExamGradeAndYear(ctx context.Context, examID uuid.UUID) (gradeID, yearID uuid.UUID, err error) {
	s, err := r.examScope(ctx, examID)
	return s.GradeLevelID, s.AcademicYearID, err
}

// examFormats loads every per-exam override for the given exams, keyed by
// exam then subject.
func (r *Repository) examFormats(ctx context.Context, examIDs []uuid.UUID) (map[uuid.UUID]map[uuid.UUID][]MarkComponent, error) {
	out := map[uuid.UUID]map[uuid.UUID][]MarkComponent{}
	if len(examIDs) == 0 {
		return out, nil
	}
	rows, err := r.pool.Query(ctx, `
		SELECT exam_id, subject_id, components FROM exam_subject_mark_formats WHERE exam_id = ANY($1)`, examIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var examID, subjectID uuid.UUID
		var raw []byte
		if err := rows.Scan(&examID, &subjectID, &raw); err != nil {
			return nil, err
		}
		var comps []MarkComponent
		if err := json.Unmarshal(raw, &comps); err != nil {
			return nil, fmt.Errorf("decode exam format: %w", err)
		}
		if out[examID] == nil {
			out[examID] = map[uuid.UUID][]MarkComponent{}
		}
		out[examID][subjectID] = comps
	}
	return out, rows.Err()
}

// EffectiveComponents is the scheme marks are entered and saved against for
// one subject in one exam: the exam's own format if it has one, otherwise the
// subject's fields. nil means one plain mark.
func (r *Repository) EffectiveComponents(ctx context.Context, examID, subjectID uuid.UUID) ([]MarkComponent, error) {
	formats, err := r.examFormats(ctx, []uuid.UUID{examID})
	if err != nil {
		return nil, err
	}
	if comps, ok := formats[examID][subjectID]; ok {
		return comps, nil
	}
	return r.GetSubjectComponents(ctx, subjectID)
}

// ListExamFormats returns every subject of the exam's grade with its
// effective scheme for that exam.
func (r *Repository) ListExamFormats(ctx context.Context, examID uuid.UUID) ([]ExamSubjectFormat, error) {
	scope, err := r.examScope(ctx, examID)
	if err != nil {
		return nil, err
	}
	subjects, err := r.ListSubjects(ctx, scope.SchoolID, scope.GradeLevelID)
	if err != nil {
		return nil, err
	}
	formats, err := r.examFormats(ctx, []uuid.UUID{examID})
	if err != nil {
		return nil, err
	}
	out := make([]ExamSubjectFormat, 0, len(subjects))
	for _, sub := range subjects {
		f := ExamSubjectFormat{SubjectID: sub.ID, SubjectName: sub.Name, MaxMarks: sub.MaxMarks}
		if comps, ok := formats[examID][sub.ID]; ok {
			f.Components, f.Custom = comps, true
		} else if f.Components, err = r.GetSubjectComponents(ctx, sub.ID); err != nil {
			return nil, err
		}
		if f.Components == nil {
			f.Components = []MarkComponent{}
		}
		out = append(out, f)
	}
	return out, nil
}

// NormalizeComponents trims labels, fills missing keys from the label and
// rejects empty labels, non-positive max marks and duplicate keys.
func NormalizeComponents(in []MarkComponent) ([]MarkComponent, error) {
	out := make([]MarkComponent, 0, len(in))
	seen := map[string]bool{}
	for _, c := range in {
		c.Label = strings.TrimSpace(c.Label)
		c.Key = strings.TrimSpace(c.Key)
		if c.Key == "" {
			c.Key = slugifyComponentKey(c.Label)
		}
		if c.Label == "" || c.Key == "" {
			return nil, fmt.Errorf("%w: every field needs a name", apperr.ErrInvalidInput)
		}
		if c.MaxMarks <= 0 {
			return nil, fmt.Errorf("%w: %s needs positive max marks", apperr.ErrInvalidInput, c.Label)
		}
		if seen[c.Key] {
			return nil, fmt.Errorf("%w: two fields are both called %q", apperr.ErrInvalidInput, c.Label)
		}
		seen[c.Key] = true
		out = append(out, c)
	}
	return out, nil
}

// ErrFormatHasMarks is returned when a format change would drop or rescale a
// field that already has marks recorded in that exam.
var ErrFormatHasMarks = errors.New("format change conflicts with recorded marks")

// checkFormatKeepsRecordedMarks refuses a new scheme for (exam, subject) that
// would drop or rescale a field with marks already recorded in that exam, or
// switch between plain and per-field entry once plain marks exist. Renames
// (same key, same max) are fine.
func (r *Repository) checkFormatKeepsRecordedMarks(ctx context.Context, examID, subjectID uuid.UUID, next []MarkComponent) error {
	rows, err := r.pool.Query(ctx, `
		SELECT DISTINCT emc.component_key FROM exam_mark_components emc
		JOIN exam_marks em ON em.id = emc.exam_mark_id
		WHERE em.exam_id = $1 AND em.subject_id = $2 AND emc.obtained <> 0`, examID, subjectID)
	if err != nil {
		return err
	}
	recorded := map[string]bool{}
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			rows.Close()
			return err
		}
		recorded[k] = true
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	current, err := r.EffectiveComponents(ctx, examID, subjectID)
	if err != nil {
		return err
	}
	curMax := map[string]int{}
	for _, c := range current {
		curMax[c.Key] = c.MaxMarks
	}
	nextMax := map[string]int{}
	for _, c := range next {
		nextMax[c.Key] = c.MaxMarks
	}
	var blocked []string
	for k := range recorded {
		m, ok := nextMax[k]
		if !ok || (curMax[k] != 0 && m != curMax[k]) {
			blocked = append(blocked, k)
		}
	}
	if len(blocked) > 0 {
		sort.Strings(blocked)
		return fmt.Errorf("%w: marks are already recorded in this exam for %s -- those fields can't be removed or have their max marks changed (renaming is fine)", ErrFormatHasMarks, strings.Join(blocked, ", "))
	}

	// Plain marks (no per-field breakdown) already entered: moving this exam to
	// per-field entry would recompute their totals from empty fields.
	if len(current) == 0 && len(next) > 0 {
		var hasPlain bool
		if err := r.pool.QueryRow(ctx, `
			SELECT EXISTS(SELECT 1 FROM exam_marks em
			WHERE em.exam_id=$1 AND em.subject_id=$2 AND em.marks_obtained <> 0
			  AND NOT EXISTS (SELECT 1 FROM exam_mark_components emc WHERE emc.exam_mark_id = em.id))`,
			examID, subjectID).Scan(&hasPlain); err != nil {
			return err
		}
		if hasPlain {
			return fmt.Errorf("%w: marks are already entered as a single number for this subject in this exam, so it can't switch to separate fields", ErrFormatHasMarks)
		}
	}
	return nil
}

func (r *Repository) subjectInExamGrade(ctx context.Context, examID, subjectID uuid.UUID) error {
	var ok bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM exams e JOIN subjects s ON s.grade_level_id = e.grade_level_id AND s.school_id = e.school_id
		WHERE e.id=$1 AND s.id=$2)`, examID, subjectID).Scan(&ok)
	if err != nil {
		return err
	}
	if !ok {
		return apperr.ErrNotFound
	}
	return nil
}

// SetExamSubjectFormat gives a subject its own marks distribution for one
// exam. components may be empty (one plain mark).
func (r *Repository) SetExamSubjectFormat(ctx context.Context, examID, subjectID uuid.UUID, components []MarkComponent, createdBy uuid.UUID) error {
	if err := r.subjectInExamGrade(ctx, examID, subjectID); err != nil {
		return err
	}
	if err := r.checkFormatKeepsRecordedMarks(ctx, examID, subjectID, components); err != nil {
		return err
	}
	if components == nil {
		components = []MarkComponent{}
	}
	raw, err := json.Marshal(components)
	if err != nil {
		return err
	}
	var by interface{}
	if createdBy != uuid.Nil {
		by = createdBy
	}
	_, err = r.pool.Exec(ctx, `
		INSERT INTO exam_subject_mark_formats (exam_id, subject_id, components, created_by)
		VALUES ($1,$2,$3,$4)
		ON CONFLICT (exam_id, subject_id) DO UPDATE SET components=$3, updated_at=NOW()`,
		examID, subjectID, raw, by)
	return database.MapError(err)
}

// ResetExamSubjectFormat drops a subject's per-exam format so its own fields
// apply again, unless that would orphan marks recorded in the exam.
func (r *Repository) ResetExamSubjectFormat(ctx context.Context, examID, subjectID uuid.UUID) error {
	if err := r.subjectInExamGrade(ctx, examID, subjectID); err != nil {
		return err
	}
	fallback, err := r.GetSubjectComponents(ctx, subjectID)
	if err != nil {
		return err
	}
	if err := r.checkFormatKeepsRecordedMarks(ctx, examID, subjectID, fallback); err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx, `DELETE FROM exam_subject_mark_formats WHERE exam_id=$1 AND subject_id=$2`, examID, subjectID)
	return err
}
