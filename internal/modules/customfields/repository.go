package customfields

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/ajaypatel01/CampusDesk/internal/domain"
	"github.com/ajaypatel01/CampusDesk/internal/platform/database"
	apperr "github.com/ajaypatel01/CampusDesk/internal/platform/errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

// NoScope is the sentinel scope_id for a definition/value that isn't scoped
// by anything further (e.g. a "student" field, unlike a "student_result"
// field which is scoped by academic year).
var NoScope = uuid.Nil

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) CreateDefinition(ctx context.Context, d *domain.CustomFieldDefinition) error {
	enumJSON, err := json.Marshal(d.EnumOptions)
	if err != nil {
		return err
	}
	row := r.pool.QueryRow(ctx, `
		INSERT INTO custom_field_definitions (school_id, entity_type, field_key, label, field_type, enum_options, sort_order, created_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		RETURNING id, created_at, updated_at`,
		d.SchoolID, d.EntityType, d.FieldKey, d.Label, d.FieldType, enumJSON, d.SortOrder, d.CreatedBy,
	)
	if err := row.Scan(&d.ID, &d.CreatedAt, &d.UpdatedAt); err != nil {
		return database.MapError(err)
	}
	return nil
}

func (r *Repository) ListDefinitions(ctx context.Context, schoolID uuid.UUID, entityType string) ([]domain.CustomFieldDefinition, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, school_id, entity_type, field_key, label, field_type, enum_options, sort_order, created_by, created_at, updated_at
		FROM custom_field_definitions WHERE school_id=$1 AND entity_type=$2
		ORDER BY sort_order, label`, schoolID, entityType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []domain.CustomFieldDefinition
	for rows.Next() {
		d, err := scanDefinition(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *d)
	}
	return items, rows.Err()
}

func (r *Repository) GetDefinition(ctx context.Context, id uuid.UUID) (*domain.CustomFieldDefinition, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, school_id, entity_type, field_key, label, field_type, enum_options, sort_order, created_by, created_at, updated_at
		FROM custom_field_definitions WHERE id=$1`, id)
	d, err := scanDefinition(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperr.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get custom field definition: %w", err)
	}
	return d, nil
}

func (r *Repository) UpdateDefinition(ctx context.Context, d *domain.CustomFieldDefinition) error {
	enumJSON, err := json.Marshal(d.EnumOptions)
	if err != nil {
		return err
	}
	tag, err := r.pool.Exec(ctx, `
		UPDATE custom_field_definitions SET label=$2, field_type=$3, enum_options=$4, sort_order=$5, updated_at=NOW()
		WHERE id=$1`, d.ID, d.Label, d.FieldType, enumJSON, d.SortOrder)
	if err != nil {
		return database.MapError(err)
	}
	if tag.RowsAffected() == 0 {
		return apperr.ErrNotFound
	}
	return nil
}

func (r *Repository) DeleteDefinition(ctx context.Context, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM custom_field_definitions WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return apperr.ErrNotFound
	}
	return nil
}

// UpsertValue sets one definition's value for one entity instance (and
// scope). An empty value string deletes the row instead of storing "" --
// clearing a field back to blank shouldn't leave a row behind.
func (r *Repository) UpsertValue(ctx context.Context, definitionID, entityID, scopeID uuid.UUID, value string) error {
	if value == "" {
		_, err := r.pool.Exec(ctx, `DELETE FROM custom_field_values WHERE definition_id=$1 AND entity_id=$2 AND scope_id=$3`,
			definitionID, entityID, scopeID)
		return err
	}
	_, err := r.pool.Exec(ctx, `
		INSERT INTO custom_field_values (definition_id, entity_id, scope_id, value_text)
		VALUES ($1,$2,$3,$4)
		ON CONFLICT (definition_id, entity_id, scope_id) DO UPDATE SET value_text=$4, updated_at=NOW()`,
		definitionID, entityID, scopeID, value,
	)
	if err != nil {
		return database.MapError(err)
	}
	return nil
}

// DefinitionWithValue pairs a definition with its current value for one
// entity instance (Value is "" when nothing has been set yet).
type DefinitionWithValue struct {
	domain.CustomFieldDefinition
	Value string `json:"value"`
}

// ListValuesForEntity returns every definition for (schoolID, entityType)
// alongside its value for (entityID, scopeID) in one call -- a definition
// with no value yet still appears, with Value == "", so the UI can render an
// empty input for it.
func (r *Repository) ListValuesForEntity(ctx context.Context, schoolID uuid.UUID, entityType string, entityID, scopeID uuid.UUID) ([]DefinitionWithValue, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT d.id, d.school_id, d.entity_type, d.field_key, d.label, d.field_type, d.enum_options, d.sort_order, d.created_by, d.created_at, d.updated_at,
			COALESCE(v.value_text, '')
		FROM custom_field_definitions d
		LEFT JOIN custom_field_values v ON v.definition_id = d.id AND v.entity_id = $3 AND v.scope_id = $4
		WHERE d.school_id = $1 AND d.entity_type = $2
		ORDER BY d.sort_order, d.label`, schoolID, entityType, entityID, scopeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []DefinitionWithValue
	for rows.Next() {
		var dv DefinitionWithValue
		var enumJSON []byte
		if err := rows.Scan(&dv.ID, &dv.SchoolID, &dv.EntityType, &dv.FieldKey, &dv.Label, &dv.FieldType, &enumJSON,
			&dv.SortOrder, &dv.CreatedBy, &dv.CreatedAt, &dv.UpdatedAt, &dv.Value); err != nil {
			return nil, err
		}
		if len(enumJSON) > 0 {
			if err := json.Unmarshal(enumJSON, &dv.EnumOptions); err != nil {
				return nil, err
			}
		}
		items = append(items, dv)
	}
	return items, rows.Err()
}

type scannable interface {
	Scan(dest ...interface{}) error
}

func scanDefinition(row scannable) (*domain.CustomFieldDefinition, error) {
	var d domain.CustomFieldDefinition
	var enumJSON []byte
	if err := row.Scan(&d.ID, &d.SchoolID, &d.EntityType, &d.FieldKey, &d.Label, &d.FieldType, &enumJSON,
		&d.SortOrder, &d.CreatedBy, &d.CreatedAt, &d.UpdatedAt); err != nil {
		return nil, err
	}
	if len(enumJSON) > 0 {
		if err := json.Unmarshal(enumJSON, &d.EnumOptions); err != nil {
			return nil, err
		}
	}
	return &d, nil
}
