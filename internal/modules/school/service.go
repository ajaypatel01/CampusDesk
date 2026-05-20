package school

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
	Name    string `json:"name"`
	Code    string `json:"code"`
	Address string `json:"address"`
	Phone   string `json:"phone"`
	Email   string `json:"email"`
}

type UpdateInput struct {
	Name    string `json:"name"`
	Code    string `json:"code"`
	Address string `json:"address"`
	Phone   string `json:"phone"`
	Email   string `json:"email"`
}

func (s *Service) Create(ctx context.Context, in CreateInput) (*domain.School, error) {
	if err := validateInput(in.Name, in.Code); err != nil {
		return nil, err
	}
	school := &domain.School{
		Name:    strings.TrimSpace(in.Name),
		Code:    strings.TrimSpace(in.Code),
		Address: strings.TrimSpace(in.Address),
		Phone:   strings.TrimSpace(in.Phone),
		Email:   strings.TrimSpace(in.Email),
	}
	if err := s.repo.Create(ctx, school); err != nil {
		return nil, err
	}
	return school, nil
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (*domain.School, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) List(ctx context.Context, limit, offset int) ([]domain.School, int, error) {
	return s.repo.List(ctx, limit, offset)
}

func (s *Service) Update(ctx context.Context, id uuid.UUID, in UpdateInput) (*domain.School, error) {
	if err := validateInput(in.Name, in.Code); err != nil {
		return nil, err
	}
	school, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	school.Name = strings.TrimSpace(in.Name)
	school.Code = strings.TrimSpace(in.Code)
	school.Address = strings.TrimSpace(in.Address)
	school.Phone = strings.TrimSpace(in.Phone)
	school.Email = strings.TrimSpace(in.Email)
	if err := s.repo.Update(ctx, school); err != nil {
		return nil, err
	}
	return s.repo.GetByID(ctx, id)
}

func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}

func validateInput(name, code string) error {
	if strings.TrimSpace(name) == "" || strings.TrimSpace(code) == "" {
		return apperr.ErrInvalidInput
	}
	return nil
}
