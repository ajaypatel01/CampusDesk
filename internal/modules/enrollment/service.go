package enrollment

import (
	"context"
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

type CreateEnrollmentInput struct {
	StudentID      uuid.UUID  `json:"student_id"`
	SchoolID       uuid.UUID  `json:"school_id"`
	AcademicYearID uuid.UUID  `json:"academic_year_id"`
	ClassSectionID *uuid.UUID `json:"class_section_id"`
	EnrollmentDate time.Time  `json:"enrollment_date"`
	Status         string     `json:"status"`
}

type UpdateEnrollmentInput struct {
	ClassSectionID *uuid.UUID `json:"class_section_id"`
	Status         string     `json:"status"`
}

type AttendanceInput struct {
	StudentID      uuid.UUID `json:"student_id"`
	SchoolID       uuid.UUID `json:"school_id"`
	ClassSectionID *uuid.UUID `json:"class_section_id"`
	RecordDate     time.Time `json:"record_date"`
	Status         string    `json:"status"`
	Notes          string    `json:"notes"`
}

func (s *Service) Create(ctx context.Context, in CreateEnrollmentInput) (*domain.Enrollment, error) {
	if in.StudentID == uuid.Nil || in.SchoolID == uuid.Nil || in.AcademicYearID == uuid.Nil {
		return nil, apperr.ErrInvalidInput
	}
	status := domain.EnrollmentStatus(in.Status)
	if status == "" {
		status = domain.EnrollmentStatusActive
	}
	enrollDate := in.EnrollmentDate
	if enrollDate.IsZero() {
		enrollDate = time.Now().UTC()
	}
	e := &domain.Enrollment{
		StudentID: in.StudentID, SchoolID: in.SchoolID, AcademicYearID: in.AcademicYearID,
		ClassSectionID: in.ClassSectionID, EnrollmentDate: enrollDate, Status: status,
	}
	if err := s.repo.Create(ctx, e); err != nil {
		return nil, err
	}
	return e, nil
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (*domain.Enrollment, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) List(ctx context.Context, schoolID, yearID uuid.UUID, limit, offset int) ([]domain.Enrollment, int, error) {
	if schoolID == uuid.Nil || yearID == uuid.Nil {
		return nil, 0, apperr.ErrInvalidInput
	}
	return s.repo.ListBySchoolYear(ctx, schoolID, yearID, limit, offset)
}

func (s *Service) Update(ctx context.Context, id uuid.UUID, in UpdateEnrollmentInput) (*domain.Enrollment, error) {
	e, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if in.ClassSectionID != nil {
		e.ClassSectionID = in.ClassSectionID
	}
	if in.Status != "" {
		e.Status = domain.EnrollmentStatus(in.Status)
	}
	if err := s.repo.Update(ctx, e); err != nil {
		return nil, err
	}
	return s.repo.GetByID(ctx, id)
}

func (s *Service) RecordAttendance(ctx context.Context, in AttendanceInput) (*domain.AttendanceRecord, error) {
	if in.StudentID == uuid.Nil || in.SchoolID == uuid.Nil || in.RecordDate.IsZero() || in.Status == "" {
		return nil, apperr.ErrInvalidInput
	}
	a := &domain.AttendanceRecord{
		StudentID: in.StudentID, SchoolID: in.SchoolID, ClassSectionID: in.ClassSectionID,
		RecordDate: in.RecordDate, Status: domain.AttendanceStatus(in.Status), Notes: in.Notes,
	}
	if err := s.repo.UpsertAttendance(ctx, a); err != nil {
		return nil, err
	}
	return a, nil
}

func (s *Service) ListAttendance(ctx context.Context, schoolID uuid.UUID, date string, sectionID *uuid.UUID) ([]domain.AttendanceRecord, error) {
	if schoolID == uuid.Nil || date == "" {
		return nil, apperr.ErrInvalidInput
	}
	return s.repo.ListAttendance(ctx, schoolID, date, sectionID)
}
