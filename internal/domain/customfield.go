package domain

import "github.com/google/uuid"

// CustomFieldDefinition is a super_admin-defined extra field on a section of
// the app (EntityType, e.g. "student" or "student_result"). New sections
// need no migration -- EntityType is free text, not a DB enum.
type CustomFieldDefinition struct {
	ID          uuid.UUID  `json:"id"`
	SchoolID    uuid.UUID  `json:"school_id"`
	EntityType  string     `json:"entity_type"`
	FieldKey    string     `json:"field_key"`
	Label       string     `json:"label"`
	FieldType   string     `json:"field_type"` // int | float | string | boolean | enum
	EnumOptions []string   `json:"enum_options,omitempty"`
	SortOrder   int        `json:"sort_order"`
	CreatedBy   *uuid.UUID `json:"created_by,omitempty"`
	Timestamps
}

// CustomFieldValue is one definition's value for one entity instance
// (e.g. one student), optionally scoped further (e.g. by academic year).
// ScopeID is the sentinel uuid.Nil-string ("00000000-...") when unscoped --
// see the migration comment for why it's never a real NULL.
type CustomFieldValue struct {
	ID           uuid.UUID `json:"id"`
	DefinitionID uuid.UUID `json:"definition_id"`
	EntityID     uuid.UUID `json:"entity_id"`
	ScopeID      uuid.UUID `json:"scope_id"`
	Value        string    `json:"value"`
	Timestamps
}
