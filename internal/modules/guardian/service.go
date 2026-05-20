package guardian

import (
	"context"
	"strings"

	"github.com/ajaypatel01/CampusDesk/internal/domain"
	apperr "github.com/ajaypatel01/CampusDesk/internal/platform/errors"
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
		FirstName: strings.TrimSpace(in.FirstName),
		LastName:  strings.TrimSpace(in.LastName),
		Email:     strings.TrimSpace(in.Email),
		Phone:     strings.TrimSpace(in.Phone),
		Relation:  strings.TrimSpace(in.Relation),
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
