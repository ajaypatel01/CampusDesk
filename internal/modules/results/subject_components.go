package results

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/ajaypatel01/CampusDesk/internal/platform/database"
	apperr "github.com/ajaypatel01/CampusDesk/internal/platform/errors"
	"github.com/google/uuid"
)

var componentSlugPattern = regexp.MustCompile(`[^a-z0-9]+`)

func slugifyComponentKey(label string) string {
	s := componentSlugPattern.ReplaceAllString(strings.ToLower(strings.TrimSpace(label)), "_")
	return strings.Trim(s, "_")
}

// This file lets an admin/registrar add graded components ("sections" --
// Oral, Unit Test, Activity, Practical, Written, or any new one) to one
// subject, independent of the fixed per-grade-template scheme in
// templates.go. A subject nobody has touched keeps using that computed
// default (nil rows here); the moment someone adds a component, this table
// becomes authoritative for that one subject only -- every other subject is
// unaffected.

// subjectGradeTemplate looks up the report-card template of a subject's own
// grade level, for computing its default (untouched) component scheme.
func (r *Repository) subjectGradeTemplate(ctx context.Context, subjectID uuid.UUID) (*string, error) {
	var template *string
	err := r.pool.QueryRow(ctx, `
		SELECT gl.report_card_template
		FROM subjects sub JOIN grade_levels gl ON gl.id = sub.grade_level_id
		WHERE sub.id = $1`, subjectID,
	).Scan(&template)
	return template, err
}

func (r *Repository) listStoredSubjectComponents(ctx context.Context, subjectID uuid.UUID) ([]MarkComponent, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT key, label, max_marks FROM subject_mark_components
		WHERE subject_id = $1 ORDER BY sort_order, label`, subjectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []MarkComponent
	for rows.Next() {
		var c MarkComponent
		if err := rows.Scan(&c.Key, &c.Label, &c.MaxMarks); err != nil {
			return nil, err
		}
		items = append(items, c)
	}
	return items, rows.Err()
}

// GetSubjectComponents returns this subject's effective component scheme:
// whatever's explicitly stored for it, or -- if nothing's been added yet --
// the same computed default every subject on its grade-template has always
// used (nil for a grade with no template, meaning plain single-number
// entry, unchanged).
func (r *Repository) GetSubjectComponents(ctx context.Context, subjectID uuid.UUID) ([]MarkComponent, error) {
	stored, err := r.listStoredSubjectComponents(ctx, subjectID)
	if err != nil {
		return nil, err
	}
	if len(stored) > 0 {
		return stored, nil
	}
	template, err := r.subjectGradeTemplate(ctx, subjectID)
	if err != nil {
		return nil, err
	}
	return MarkComponentsForTemplate(template), nil
}

// AddSubjectComponent appends one new component to a subject. On that
// subject's first-ever custom component, its current computed default is
// persisted first (so nothing already recorded under those keys silently
// changes meaning), then the new one is appended after it.
func (r *Repository) AddSubjectComponent(ctx context.Context, subjectID uuid.UUID, key, label string, maxMarks int, createdBy uuid.UUID) ([]MarkComponent, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var count int
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM subject_mark_components WHERE subject_id=$1`, subjectID).Scan(&count); err != nil {
		return nil, err
	}
	nextSort := count
	if count == 0 {
		template, err := r.subjectGradeTemplate(ctx, subjectID)
		if err != nil {
			return nil, err
		}
		for i, c := range MarkComponentsForTemplate(template) {
			if _, err := tx.Exec(ctx, `
				INSERT INTO subject_mark_components (subject_id, key, label, max_marks, sort_order, created_by)
				VALUES ($1,$2,$3,$4,$5,$6)`,
				subjectID, c.Key, c.Label, c.MaxMarks, i, createdBy,
			); err != nil {
				return nil, database.MapError(err)
			}
			nextSort = i + 1
		}
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO subject_mark_components (subject_id, key, label, max_marks, sort_order, created_by)
		VALUES ($1,$2,$3,$4,$5,$6)`,
		subjectID, key, label, maxMarks, nextSort, createdBy,
	); err != nil {
		return nil, database.MapError(err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return r.listStoredSubjectComponents(ctx, subjectID)
}

// DeleteSubjectComponent removes one component from a subject -- refused if
// any exam already has a recorded (non-zero) value under that key, so
// deleting a component can't silently make real marks disappear.
//
// A subject nobody has customised has no stored rows: its components are the
// grade template's computed default. Removing one of those first persists
// the rest of the default (same as AddSubjectComponent does on a first add),
// otherwise there'd be nothing to delete and the field could never go away.
func (r *Repository) DeleteSubjectComponent(ctx context.Context, subjectID uuid.UUID, key string, createdBy uuid.UUID) error {
	var hasValues bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM exam_mark_components emc
			JOIN exam_marks em ON em.id = emc.exam_mark_id
			WHERE em.subject_id = $1 AND emc.component_key = $2 AND emc.obtained <> 0
		)`, subjectID, key,
	).Scan(&hasValues)
	if err != nil {
		return err
	}
	if hasValues {
		return apperr.ErrConflict
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var count int
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM subject_mark_components WHERE subject_id=$1`, subjectID).Scan(&count); err != nil {
		return err
	}
	template, err := r.subjectGradeTemplate(ctx, subjectID)
	if err != nil {
		return err
	}
	defaults := MarkComponentsForTemplate(template)

	if count == 0 {
		found := false
		for _, c := range defaults {
			if c.Key == key {
				found = true
			}
		}
		if !found {
			return apperr.ErrNotFound
		}
		if len(defaults) == 1 {
			return lastTemplateFieldErr
		}
		sort := 0
		for _, c := range defaults {
			if c.Key == key {
				continue
			}
			if _, err := tx.Exec(ctx, `
				INSERT INTO subject_mark_components (subject_id, key, label, max_marks, sort_order, created_by)
				VALUES ($1,$2,$3,$4,$5,$6)`,
				subjectID, c.Key, c.Label, c.MaxMarks, sort, createdBy,
			); err != nil {
				return database.MapError(err)
			}
			sort++
		}
		return tx.Commit(ctx)
	}

	// With stored rows, removing the last one would make the subject fall
	// back to its template default -- the field would come straight back.
	if count == 1 && len(defaults) > 0 {
		return lastTemplateFieldErr
	}
	tag, err := tx.Exec(ctx, `DELETE FROM subject_mark_components WHERE subject_id=$1 AND key=$2`, subjectID, key)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return apperr.ErrNotFound
	}
	return tx.Commit(ctx)
}

var lastTemplateFieldErr = fmt.Errorf("%w: a subject on this grade's report-card template needs at least one field -- add the new field first, then remove this one", apperr.ErrInvalidInput)
