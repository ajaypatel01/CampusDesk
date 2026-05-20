package academic

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

type YearInput struct {
	SchoolID  uuid.UUID `json:"school_id"`
	Name      string    `json:"name"`
	StartDate time.Time `json:"start_date"`
	EndDate   time.Time `json:"end_date"`
	IsCurrent bool      `json:"is_current"`
}

type GradeInput struct {
	SchoolID  uuid.UUID `json:"school_id"`
	Name      string    `json:"name"`
	SortOrder int       `json:"sort_order"`
}

type SectionInput struct {
	SchoolID          uuid.UUID  `json:"school_id"`
	AcademicYearID    uuid.UUID  `json:"academic_year_id"`
	GradeLevelID      uuid.UUID  `json:"grade_level_id"`
	Name              string     `json:"name"`
	Capacity          int        `json:"capacity"`
	HomeroomTeacherID *uuid.UUID `json:"homeroom_teacher_id"`
}

func (s *Service) CreateYear(ctx context.Context, in YearInput) (*domain.AcademicYear, error) {
	if in.SchoolID == uuid.Nil || strings.TrimSpace(in.Name) == "" || !in.EndDate.After(in.StartDate) {
		return nil, apperr.ErrInvalidInput
	}
	y := &domain.AcademicYear{
		SchoolID: in.SchoolID, Name: strings.TrimSpace(in.Name),
		StartDate: in.StartDate, EndDate: in.EndDate, IsCurrent: in.IsCurrent,
	}
	if err := s.repo.CreateYear(ctx, y); err != nil {
		return nil, err
	}
	return y, nil
}

func (s *Service) ListYears(ctx context.Context, schoolID uuid.UUID) ([]domain.AcademicYear, error) {
	if schoolID == uuid.Nil {
		return nil, apperr.ErrInvalidInput
	}
	return s.repo.ListYears(ctx, schoolID)
}

func (s *Service) CreateGrade(ctx context.Context, in GradeInput) (*domain.GradeLevel, error) {
	if in.SchoolID == uuid.Nil || strings.TrimSpace(in.Name) == "" {
		return nil, apperr.ErrInvalidInput
	}
	g := &domain.GradeLevel{SchoolID: in.SchoolID, Name: strings.TrimSpace(in.Name), SortOrder: in.SortOrder}
	if err := s.repo.CreateGrade(ctx, g); err != nil {
		return nil, err
	}
	return g, nil
}

func (s *Service) ListGrades(ctx context.Context, schoolID uuid.UUID) ([]domain.GradeLevel, error) {
	if schoolID == uuid.Nil {
		return nil, apperr.ErrInvalidInput
	}
	return s.repo.ListGrades(ctx, schoolID)
}

func (s *Service) CreateSection(ctx context.Context, in SectionInput) (*domain.ClassSection, error) {
	if in.SchoolID == uuid.Nil || in.AcademicYearID == uuid.Nil || in.GradeLevelID == uuid.Nil || strings.TrimSpace(in.Name) == "" {
		return nil, apperr.ErrInvalidInput
	}
	cap := in.Capacity
	if cap <= 0 {
		cap = 30
	}
	c := &domain.ClassSection{
		SchoolID: in.SchoolID, AcademicYearID: in.AcademicYearID, GradeLevelID: in.GradeLevelID,
		Name: strings.TrimSpace(in.Name), Capacity: cap, HomeroomTeacherID: in.HomeroomTeacherID,
	}
	if err := s.repo.CreateSection(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

func (s *Service) ListSections(ctx context.Context, schoolID, yearID uuid.UUID) ([]domain.ClassSection, error) {
	if schoolID == uuid.Nil || yearID == uuid.Nil {
		return nil, apperr.ErrInvalidInput
	}
	return s.repo.ListSections(ctx, schoolID, yearID)
}
