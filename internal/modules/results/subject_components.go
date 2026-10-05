package results

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/ajaypatel01/CampusDesk/internal/platform/database"
	apperr "github.com/ajaypatel01/CampusDesk/internal/platform/errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v4"
)

var componentSlugPattern = regexp.MustCompile(`[^a-z0-9]+`)

func slugifyComponentKey(label string) string {
	s := componentSlugPattern.ReplaceAllString(strings.ToLower(strings.TrimSpace(label)), "_")
	return strings.Trim(s, "_")
}

// This file lets an admin/registrar add, edit and remove graded components
// ("sections" -- Oral, Unit Test, Activity, Practical, Written, or any new
// one) on one subject, independent of the fixed per-grade-template scheme in
// templates.go. A subject nobody has touched has no rows here: subjects that
// predate per-subject fields (uses_template_components) keep using the
// template's computed default, while subjects created since start with no
// fields at all. The moment someone changes a subject's fields, this table
// becomes authoritative for that one subject only.

// subjectDefaultComponents is a subject's scheme while it has no stored
// rows: its grade template's fields if it uses them, otherwise none (plain
// single-number entry).
func (r *Repository) subjectDefaultComponents(ctx context.Context, subjectID uuid.UUID) ([]MarkComponent, error) {
	var template *string
	var usesTemplate bool
	err := r.pool.QueryRow(ctx, `
		SELECT gl.report_card_template, sub.uses_template_components
		FROM subjects sub JOIN grade_levels gl ON gl.id = sub.grade_level_id
		WHERE sub.id = $1`, subjectID,
	).Scan(&template, &usesTemplate)
	if err != nil {
		return nil, err
	}
	if !usesTemplate {
		return nil, nil
	}
	return MarkComponentsForTemplate(template), nil
}

// persistDefaultComponents writes a subject's default scheme as stored rows,
// skipping skipKey (pass "" to keep all), so the subject can then be edited
// row by row. Returns how many rows it wrote.
func persistDefaultComponents(ctx context.Context, tx pgx.Tx, subjectID uuid.UUID, defaults []MarkComponent, skipKey string, createdBy uuid.UUID) (int, error) {
	n := 0
	for _, c := range defaults {
		if c.Key == skipKey {
			continue
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO subject_mark_components (subject_id, key, label, max_marks, sort_order, created_by)
			VALUES ($1,$2,$3,$4,$5,$6)`,
			subjectID, c.Key, c.Label, c.MaxMarks, n, createdBy,
		); err != nil {
			return n, database.MapError(err)
		}
		n++
	}
	return n, nil
}

// componentHasRecordedMarks reports whether any exam has a non-zero value
// recorded under this subject's component key.
func (r *Repository) componentHasRecordedMarks(ctx context.Context, subjectID uuid.UUID, key string) (bool, error) {
	var has bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM exam_mark_components emc
			JOIN exam_marks em ON em.id = emc.exam_mark_id
			WHERE em.subject_id = $1 AND emc.component_key = $2 AND emc.obtained <> 0
		)`, subjectID, key,
	).Scan(&has)
	return has, err
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
// whatever's explicitly stored for it, or -- if nothing's been changed yet --
// its default (subjectDefaultComponents; nil means plain single-number entry).
func (r *Repository) GetSubjectComponents(ctx context.Context, subjectID uuid.UUID) ([]MarkComponent, error) {
	stored, err := r.listStoredSubjectComponents(ctx, subjectID)
	if err != nil {
		return nil, err
	}
	if len(stored) > 0 {
		return stored, nil
	}
	return r.subjectDefaultComponents(ctx, subjectID)
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
		defaults, err := r.subjectDefaultComponents(ctx, subjectID)
		if err != nil {
			return nil, err
		}
		if nextSort, err = persistDefaultComponents(ctx, tx, subjectID, defaults, "", createdBy); err != nil {
			return nil, err
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
	hasValues, err := r.componentHasRecordedMarks(ctx, subjectID, key)
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
	defaults, err := r.subjectDefaultComponents(ctx, subjectID)
	if err != nil {
		return err
	}

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
		if _, err := persistDefaultComponents(ctx, tx, subjectID, defaults, key, createdBy); err != nil {
			return err
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

// UpdateSubjectComponent changes one component's label and/or max marks. A
// subject still on its default scheme has it persisted first, so the change
// sticks. Changing max marks is refused once marks are recorded under the
// key: existing marks were entered on the old scale and would no longer add
// up. Renaming is always allowed.
func (r *Repository) UpdateSubjectComponent(ctx context.Context, subjectID uuid.UUID, key, label string, maxMarks int, createdBy uuid.UUID) ([]MarkComponent, error) {
	current, err := r.GetSubjectComponents(ctx, subjectID)
	if err != nil {
		return nil, err
	}
	var existing *MarkComponent
	for i := range current {
		if current[i].Key == key {
			existing = &current[i]
		}
	}
	if existing == nil {
		return nil, apperr.ErrNotFound
	}
	if label == "" {
		label = existing.Label
	}
	if maxMarks == 0 {
		maxMarks = existing.MaxMarks
	}
	if maxMarks != existing.MaxMarks {
		hasValues, err := r.componentHasRecordedMarks(ctx, subjectID, key)
		if err != nil {
			return nil, err
		}
		if hasValues {
			return nil, apperr.ErrConflict
		}
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var count int
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM subject_mark_components WHERE subject_id=$1`, subjectID).Scan(&count); err != nil {
		return nil, err
	}
	if count == 0 {
		if _, err := persistDefaultComponents(ctx, tx, subjectID, current, "", createdBy); err != nil {
			return nil, err
		}
	}
	if _, err := tx.Exec(ctx, `
		UPDATE subject_mark_components SET label=$3, max_marks=$4
		WHERE subject_id=$1 AND key=$2`, subjectID, key, label, maxMarks,
	); err != nil {
		return nil, database.MapError(err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return r.listStoredSubjectComponents(ctx, subjectID)
}
