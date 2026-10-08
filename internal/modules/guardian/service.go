package guardian

import (
	"context"
	"strings"

	"github.com/ajaypatel01/CampusDesk/internal/domain"
	apperr "github.com/ajaypatel01/CampusDesk/internal/platform/errors"
	"github.com/ajaypatel01/CampusDesk/internal/platform/httpx"
	"github.com/google/uuid"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

type CreateInput struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
	Phone     string `json:"phone"`
	Relation  string `json:"relation"`
	Aadhar    string `json:"aadhar_number"`
}

type LinkInput struct {
	StudentID  uuid.UUID `json:"student_id"`
	GuardianID uuid.UUID `json:"guardian_id"`
	IsPrimary  bool      `json:"is_primary"`
}

func (s *Service) Create(ctx context.Context, in CreateInput) (*domain.Guardian, error) {
	if strings.TrimSpace(in.FirstName) == "" || strings.TrimSpace(in.LastName) == "" {
		return nil, apperr.ErrInvalidInput
	}
	g := &domain.Guardian{
		FirstName:    strings.TrimSpace(in.FirstName),
		LastName:     strings.TrimSpace(in.LastName),
		Email:        strings.TrimSpace(in.Email),
		Phone:        strings.TrimSpace(in.Phone),
		Relation:     strings.TrimSpace(in.Relation),
		AadharNumber: strings.TrimSpace(in.Aadhar),
	}
	if err := s.repo.Create(ctx, g); err != nil {
		return nil, err
	}
	return g, nil
}

func (s *Service) Link(ctx context.Context, in LinkInput) error {
	if in.StudentID == uuid.Nil || in.GuardianID == uuid.Nil {
		return apperr.ErrInvalidInput
	}
	return s.repo.LinkStudent(ctx, in.StudentID, in.GuardianID, in.IsPrimary)
}

func (s *Service) ListByStudent(ctx context.Context, studentID uuid.UUID) ([]domain.Guardian, error) {
	if studentID == uuid.Nil {
		return nil, apperr.ErrInvalidInput
	}
	return s.repo.ListByStudent(ctx, studentID)
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (*domain.Guardian, error) {
	return s.repo.GetByID(ctx, id)
}

// Update replaces a guardian's details (same fields and rules as Create).
func (s *Service) Update(ctx context.Context, id uuid.UUID, in CreateInput) (*domain.Guardian, error) {
	if strings.TrimSpace(in.FirstName) == "" || strings.TrimSpace(in.LastName) == "" {
		return nil, apperr.ErrInvalidInput
	}
	g := &domain.Guardian{
		ID:           id,
		FirstName:    strings.TrimSpace(in.FirstName),
		LastName:     strings.TrimSpace(in.LastName),
		Email:        strings.TrimSpace(in.Email),
		Phone:        strings.TrimSpace(in.Phone),
		Relation:     strings.TrimSpace(in.Relation),
		AadharNumber: strings.TrimSpace(in.Aadhar),
	}
	if err := s.repo.Update(ctx, g); err != nil {
		return nil, err
	}
	return g, nil
}

// callerSchool returns the school the caller is limited to, or nil for a
// super_admin, who works across schools.
func callerSchool(claims *httpx.Claims) (*uuid.UUID, error) {
	if claims == nil {
		return nil, apperr.ErrUnauthorized
	}
	if claims.Role == string(domain.RoleSuperAdmin) {
		return nil, nil
	}
	id, err := uuid.Parse(claims.SchoolID)
	if err != nil {
		return nil, apperr.ErrForbidden
	}
	return &id, nil
}

// CheckStudent fails with not-found unless the caller may see this
// student's guardians: the student must be in the caller's school.
func (s *Service) CheckStudent(ctx context.Context, claims *httpx.Claims, studentID uuid.UUID) error {
	school, err := callerSchool(claims)
	if err != nil || school == nil {
		return err
	}
	sid, err := s.repo.StudentSchoolID(ctx, studentID)
	if err != nil {
		return err
	}
	if sid != *school {
		return apperr.ErrNotFound
	}
	return nil
}

// CheckGuardian fails with not-found unless the guardian is linked to a
// student in the caller's school. allowUnlinked also admits a guardian not
// linked to anyone yet -- one just created, about to be linked.
func (s *Service) CheckGuardian(ctx context.Context, claims *httpx.Claims, guardianID uuid.UUID, allowUnlinked bool) error {
	school, err := callerSchool(claims)
	if err != nil || school == nil {
		return err
	}
	inSchool, linked, err := s.repo.GuardianSchools(ctx, guardianID, *school)
	if err != nil {
		return err
	}
	if inSchool || (allowUnlinked && !linked) {
		return nil
	}
	return apperr.ErrNotFound
}
