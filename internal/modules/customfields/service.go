package customfields

import (
	"context"
	"regexp"
	"strconv"
	"strings"

	"github.com/ajaypatel01/CampusDesk/internal/domain"
	apperr "github.com/ajaypatel01/CampusDesk/internal/platform/errors"
	"github.com/google/uuid"
)

// allowedEntityTypes is deliberately small and explicit: a typo in a request
// (e.g. "studnet") would otherwise silently create a definition no page ever
// renders. Add to this list when a new page starts using CustomFieldsSection.
var allowedEntityTypes = map[string]bool{
	"student":        true,
	"student_result": true,
}

var allowedFieldTypes = map[string]bool{
	"int": true, "float": true, "string": true, "boolean": true, "enum": true,
}

var slugPattern = regexp.MustCompile(`[^a-z0-9]+`)

func slugify(label string) string {
	s := slugPattern.ReplaceAllString(strings.ToLower(strings.TrimSpace(label)), "_")
	return strings.Trim(s, "_")
}

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

type DefinitionInput struct {
	SchoolID    uuid.UUID `json:"school_id"`
	EntityType  string    `json:"entity_type"`
	FieldKey    string    `json:"field_key"`
	Label       string    `json:"label"`
	FieldType   string    `json:"field_type"`
	EnumOptions []string  `json:"enum_options"`
	SortOrder   int       `json:"sort_order"`
}

func (s *Service) CreateDefinition(ctx context.Context, in DefinitionInput, createdBy uuid.UUID) (*domain.CustomFieldDefinition, error) {
	if in.SchoolID == uuid.Nil || strings.TrimSpace(in.Label) == "" {
		return nil, apperr.ErrInvalidInput
	}
	if !allowedEntityTypes[in.EntityType] {
		return nil, apperr.ErrInvalidInput
	}
	if !allowedFieldTypes[in.FieldType] {
		return nil, apperr.ErrInvalidInput
	}
	if in.FieldType == "enum" && len(in.EnumOptions) == 0 {
		return nil, apperr.ErrInvalidInput
	}
	key := slugify(in.FieldKey)
	if key == "" {
		key = slugify(in.Label)
	}
	if key == "" {
		return nil, apperr.ErrInvalidInput
	}
	d := &domain.CustomFieldDefinition{
		SchoolID: in.SchoolID, EntityType: in.EntityType, FieldKey: key, Label: strings.TrimSpace(in.Label),
		FieldType: in.FieldType, EnumOptions: in.EnumOptions, SortOrder: in.SortOrder, CreatedBy: &createdBy,
	}
	if err := s.repo.CreateDefinition(ctx, d); err != nil {
		return nil, err
	}
	return d, nil
}

func (s *Service) ListDefinitions(ctx context.Context, schoolID uuid.UUID, entityType string) ([]domain.CustomFieldDefinition, error) {
	if schoolID == uuid.Nil || !allowedEntityTypes[entityType] {
		return nil, apperr.ErrInvalidInput
	}
	return s.repo.ListDefinitions(ctx, schoolID, entityType)
}

type UpdateDefinitionInput struct {
	Label       string   `json:"label"`
	FieldType   string   `json:"field_type"`
	EnumOptions []string `json:"enum_options"`
	SortOrder   int      `json:"sort_order"`
}

func (s *Service) UpdateDefinition(ctx context.Context, id uuid.UUID, in UpdateDefinitionInput) (*domain.CustomFieldDefinition, error) {
	if strings.TrimSpace(in.Label) == "" || !allowedFieldTypes[in.FieldType] {
		return nil, apperr.ErrInvalidInput
	}
	if in.FieldType == "enum" && len(in.EnumOptions) == 0 {
		return nil, apperr.ErrInvalidInput
	}
	d, err := s.repo.GetDefinition(ctx, id)
	if err != nil {
		return nil, err
	}
	d.Label = strings.TrimSpace(in.Label)
	d.FieldType = in.FieldType
	d.EnumOptions = in.EnumOptions
	d.SortOrder = in.SortOrder
	if err := s.repo.UpdateDefinition(ctx, d); err != nil {
		return nil, err
	}
	return d, nil
}

func (s *Service) DeleteDefinition(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteDefinition(ctx, id)
}

func (s *Service) ListValuesForEntity(ctx context.Context, schoolID uuid.UUID, entityType string, entityID, scopeID uuid.UUID) ([]DefinitionWithValue, error) {
	if schoolID == uuid.Nil || entityID == uuid.Nil || !allowedEntityTypes[entityType] {
		return nil, apperr.ErrInvalidInput
	}
	return s.repo.ListValuesForEntity(ctx, schoolID, entityType, entityID, scopeID)
}

type UpsertValueInput struct {
	DefinitionID uuid.UUID `json:"definition_id"`
	EntityID     uuid.UUID `json:"entity_id"`
	ScopeID      uuid.UUID `json:"scope_id"`
	Value        string    `json:"value"`
}

// UpsertValue validates the submitted value against its definition's
// field_type before storing it -- an "int" field can't end up holding
// "abc", an "enum" field can't end up holding an option that was never
// defined.
func (s *Service) UpsertValue(ctx context.Context, in UpsertValueInput) error {
	if in.DefinitionID == uuid.Nil || in.EntityID == uuid.Nil {
		return apperr.ErrInvalidInput
	}
	def, err := s.repo.GetDefinition(ctx, in.DefinitionID)
	if err != nil {
		return err
	}
	value := strings.TrimSpace(in.Value)
	if value != "" {
		switch def.FieldType {
		case "int":
			if _, err := strconv.ParseInt(value, 10, 64); err != nil {
				return apperr.ErrInvalidInput
			}
		case "float":
			if _, err := strconv.ParseFloat(value, 64); err != nil {
				return apperr.ErrInvalidInput
			}
		case "boolean":
			if value != "true" && value != "false" {
				return apperr.ErrInvalidInput
			}
		case "enum":
			valid := false
			for _, opt := range def.EnumOptions {
				if opt == value {
					valid = true
					break
				}
			}
			if !valid {
				return apperr.ErrInvalidInput
			}
		}
	}
	return s.repo.UpsertValue(ctx, in.DefinitionID, in.EntityID, in.ScopeID, value)
}
