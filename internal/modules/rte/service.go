package rte

import (
	"context"

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

type RTEQuotaDetail struct {
	domain.RTEQuota
	GradeLevelName     string `json:"grade_level_name"`
	UtilizedSeats      int    `json:"utilized_seats"`
	AvailableSeats     int    `json:"available_seats"`
	TotalReimbursement int    `json:"total_reimbursement"`
}

type UpsertQuotaInput struct {
	SchoolID                    uuid.UUID `json:"school_id"`
	AcademicYearID              uuid.UUID `json:"academic_year_id"`
	GradeLevelID                uuid.UUID `json:"grade_level_id"`
	TotalSeats                  int       `json:"total_seats"`
	GovtReimbursementPerStudent int       `json:"govt_reimbursement_per_student"`
	Notes                       string    `json:"notes"`
}

func (s *Service) UpsertQuota(ctx context.Context, in UpsertQuotaInput) (*domain.RTEQuota, error) {
	if in.SchoolID == uuid.Nil || in.AcademicYearID == uuid.Nil || in.GradeLevelID == uuid.Nil {
		return nil, apperr.ErrInvalidInput
	}
	if in.TotalSeats < 0 {
		return nil, apperr.ErrInvalidInput
	}
	q := &domain.RTEQuota{
		SchoolID: in.SchoolID, AcademicYearID: in.AcademicYearID, GradeLevelID: in.GradeLevelID,
		TotalSeats: in.TotalSeats, GovtReimbursementPerStudent: in.GovtReimbursementPerStudent,
		Notes: in.Notes,
	}
	if err := s.repo.UpsertQuota(ctx, q); err != nil {
		return nil, err
	}
	return q, nil
}

func (s *Service) ListQuotas(ctx context.Context, schoolID, yearID uuid.UUID) ([]RTEQuotaDetail, error) {
	if schoolID == uuid.Nil || yearID == uuid.Nil {
		return nil, apperr.ErrInvalidInput
	}
	return s.repo.ListQuotas(ctx, schoolID, yearID)
}

func (s *Service) DeleteQuota(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteQuota(ctx, id)
}

func (s *Service) ListRTEStudents(ctx context.Context, schoolID, yearID uuid.UUID) ([]RTEStudent, error) {
	if schoolID == uuid.Nil || yearID == uuid.Nil {
		return nil, apperr.ErrInvalidInput
	}
	return s.repo.ListRTEStudents(ctx, schoolID, yearID)
}

func (s *Service) GetSummary(ctx context.Context, schoolID, yearID uuid.UUID) (*RTESummary, error) {
	if schoolID == uuid.Nil || yearID == uuid.Nil {
		return nil, apperr.ErrInvalidInput
	}
	return s.repo.GetSummary(ctx, schoolID, yearID)
}
