package student

import (
	"context"
	"strings"
	"time"

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
	SchoolID    uuid.UUID  `json:"school_id"`
	StudentCode string     `json:"student_code"`
	FirstName   string     `json:"first_name"`
	LastName    string     `json:"last_name"`
	DateOfBirth *time.Time `json:"date_of_birth"`
	Gender      string     `json:"gender"`
	Email       string     `json:"email"`
	Phone       string     `json:"phone"`
	Address     string     `json:"address"`
	Status      string     `json:"status"`
}

type UpdateInput struct {
	StudentCode string     `json:"student_code"`
	FirstName   string     `json:"first_name"`
	LastName    string     `json:"last_name"`
	DateOfBirth *time.Time `json:"date_of_birth"`
	Gender      string     `json:"gender"`
	Email       string     `json:"email"`
	Phone       string     `json:"phone"`
	Address     string     `json:"address"`
	Status      string     `json:"status"`
}

func (s *Service) Create(ctx context.Context, in CreateInput) (*domain.Student, error) {
	if in.SchoolID == uuid.Nil || strings.TrimSpace(in.StudentCode) == "" ||
		strings.TrimSpace(in.FirstName) == "" || strings.TrimSpace(in.LastName) == "" {
		return nil, apperr.ErrInvalidInput
	}
	status := domain.StudentStatus(in.Status)
	if status == "" {
		status = domain.StudentStatusActive
	}
	st := &domain.Student{
		SchoolID:    in.SchoolID,
		StudentCode: strings.TrimSpace(in.StudentCode),
		FirstName:   strings.TrimSpace(in.FirstName),
		LastName:    strings.TrimSpace(in.LastName),
		DateOfBirth: in.DateOfBirth,
		Gender:      strings.TrimSpace(in.Gender),
		Email:       strings.TrimSpace(in.Email),
		Phone:       strings.TrimSpace(in.Phone),
		Address:     strings.TrimSpace(in.Address),
		Status:      status,
	}
	if err := s.repo.Create(ctx, st); err != nil {
		return nil, err
	}
	return st, nil
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (*domain.Student, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) List(ctx context.Context, f ListFilter, limit, offset int) ([]domain.Student, int, error) {
	if f.SchoolID == uuid.Nil {
		return nil, 0, apperr.ErrInvalidInput
	}
	return s.repo.List(ctx, f, limit, offset)
}

func (s *Service) Update(ctx context.Context, id uuid.UUID, in UpdateInput) (*domain.Student, error) {
	st, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(in.StudentCode) == "" || strings.TrimSpace(in.FirstName) == "" || strings.TrimSpace(in.LastName) == "" {
		return nil, apperr.ErrInvalidInput
	}
	st.StudentCode = strings.TrimSpace(in.StudentCode)
	st.FirstName = strings.TrimSpace(in.FirstName)
	st.LastName = strings.TrimSpace(in.LastName)
	st.DateOfBirth = in.DateOfBirth
	st.Gender = strings.TrimSpace(in.Gender)
	st.Email = strings.TrimSpace(in.Email)
	st.Phone = strings.TrimSpace(in.Phone)
	st.Address = strings.TrimSpace(in.Address)
	if in.Status != "" {
		st.Status = domain.StudentStatus(in.Status)
	}
	if err := s.repo.Update(ctx, st); err != nil {
		return nil, err
	}
	return s.repo.GetByID(ctx, id)
}

func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}
