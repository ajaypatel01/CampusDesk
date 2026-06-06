package result

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
	StudentID      uuid.UUID      `json:"student_id"`
	SchoolID       uuid.UUID      `json:"school_id"`
	AcademicYearID uuid.UUID      `json:"academic_year_id"`
	ClassSectionID *uuid.UUID     `json:"class_section_id"`
	ExamName       string         `json:"exam_name"`
	FinalGrade     string         `json:"final_grade"`
	Remarks        string         `json:"remarks"`
	ResultDate     time.Time      `json:"result_date"`
	Status         string         `json:"status"`
	Subjects       []SubjectInput `json:"subjects"`
}

type UpdateInput struct {
	ClassSectionID *uuid.UUID     `json:"class_section_id"`
	ExamName       string         `json:"exam_name"`
	FinalGrade     string         `json:"final_grade"`
	Remarks        string         `json:"remarks"`
	ResultDate     time.Time      `json:"result_date"`
	Status         string         `json:"status"`
	Subjects       []SubjectInput `json:"subjects"`
}

type SubjectInput struct {
	SubjectName   string  `json:"subject_name"`
	MarksObtained float64 `json:"marks_obtained"`
	MaxMarks      float64 `json:"max_marks"`
	Grade         string  `json:"grade"`
	Remarks       string  `json:"remarks"`
}

func (s *Service) Create(ctx context.Context, in CreateInput) (*domain.Result, error) {
	if err := validateHeader(in.StudentID, in.SchoolID, in.AcademicYearID, in.ExamName); err != nil {
		return nil, err
	}
	subjects, totalMarks, maxTotalMarks, err := buildSubjects(in.Subjects)
	if err != nil {
		return nil, err
	}
	percentage := calculatePercentage(totalMarks, maxTotalMarks)
	status := domain.ResultStatus(in.Status)
	if status == "" {
		status = domain.ResultStatusDraft
	}
	resultDate := in.ResultDate
	if resultDate.IsZero() {
		resultDate = time.Now().UTC()
	}
	finalGrade := strings.TrimSpace(in.FinalGrade)
	if finalGrade == "" {
		finalGrade = gradeFromPercentage(percentage)
	}

	res := &domain.Result{
		StudentID:      in.StudentID,
		SchoolID:       in.SchoolID,
		AcademicYearID: in.AcademicYearID,
		ClassSectionID: in.ClassSectionID,
		ExamName:       strings.TrimSpace(in.ExamName),
		TotalMarks:     totalMarks,
		MaxTotalMarks:  maxTotalMarks,
		Percentage:     percentage,
		FinalGrade:     finalGrade,
		Remarks:        strings.TrimSpace(in.Remarks),
		ResultDate:     resultDate,
		Status:         status,
		Subjects:       subjects,
	}
	if err := s.repo.Create(ctx, res); err != nil {
		return nil, err
	}
	return res, nil
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (*domain.Result, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) List(ctx context.Context, filter ListFilter, limit, offset int) ([]domain.Result, int, error) {
	if filter.SchoolID == uuid.Nil || filter.AcademicYearID == uuid.Nil {
		return nil, 0, apperr.ErrInvalidInput
	}
	return s.repo.List(ctx, filter, limit, offset)
}

func (s *Service) Update(ctx context.Context, id uuid.UUID, in UpdateInput) (*domain.Result, error) {
	res, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := validateHeader(res.StudentID, res.SchoolID, res.AcademicYearID, in.ExamName); err != nil {
		return nil, err
	}
	subjects, totalMarks, maxTotalMarks, err := buildSubjects(in.Subjects)
	if err != nil {
		return nil, err
	}
	percentage := calculatePercentage(totalMarks, maxTotalMarks)
	finalGrade := strings.TrimSpace(in.FinalGrade)
	if finalGrade == "" {
		finalGrade = gradeFromPercentage(percentage)
	}

	res.ClassSectionID = in.ClassSectionID
	res.ExamName = strings.TrimSpace(in.ExamName)
	res.TotalMarks = totalMarks
	res.MaxTotalMarks = maxTotalMarks
	res.Percentage = percentage
	res.FinalGrade = finalGrade
	res.Remarks = strings.TrimSpace(in.Remarks)
	res.Subjects = subjects
	if !in.ResultDate.IsZero() {
		res.ResultDate = in.ResultDate
	}
	if in.Status != "" {
		res.Status = domain.ResultStatus(in.Status)
	}
	if err := s.repo.Update(ctx, res); err != nil {
		return nil, err
	}
	return s.repo.GetByID(ctx, id)
}

func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}

func validateHeader(studentID, schoolID, yearID uuid.UUID, examName string) error {
	if studentID == uuid.Nil || schoolID == uuid.Nil || yearID == uuid.Nil {
		return apperr.ErrInvalidInput
	}
	if strings.TrimSpace(examName) == "" {
		return apperr.ErrInvalidInput
	}
	return nil
}

func buildSubjects(in []SubjectInput) ([]domain.ResultSubject, float64, float64, error) {
	if len(in) == 0 {
		return nil, 0, 0, apperr.ErrInvalidInput
	}

	seen := make(map[string]struct{}, len(in))
	subjects := make([]domain.ResultSubject, 0, len(in))
	var totalMarks float64
	var maxTotalMarks float64

	for i, subject := range in {
		name := strings.TrimSpace(subject.SubjectName)
		key := strings.ToLower(name)
		if name == "" || subject.MaxMarks <= 0 || subject.MarksObtained < 0 || subject.MarksObtained > subject.MaxMarks {
			return nil, 0, 0, apperr.ErrInvalidInput
		}
		if _, ok := seen[key]; ok {
			return nil, 0, 0, apperr.ErrInvalidInput
		}
		seen[key] = struct{}{}

		grade := strings.TrimSpace(subject.Grade)
		if grade == "" {
			grade = gradeFromPercentage(calculatePercentage(subject.MarksObtained, subject.MaxMarks))
		}
		subjects = append(subjects, domain.ResultSubject{
			SubjectName:   name,
			MarksObtained: subject.MarksObtained,
			MaxMarks:      subject.MaxMarks,
			Grade:         grade,
			Remarks:       strings.TrimSpace(subject.Remarks),
			SortOrder:     i + 1,
		})
		totalMarks += subject.MarksObtained
		maxTotalMarks += subject.MaxMarks
	}

	return subjects, totalMarks, maxTotalMarks, nil
}

func calculatePercentage(marks, maxMarks float64) float64 {
	if maxMarks <= 0 {
		return 0
	}
	return marks / maxMarks * 100
}

func gradeFromPercentage(percentage float64) string {
	switch {
	case percentage >= 90:
		return "A+"
	case percentage >= 80:
		return "A"
	case percentage >= 70:
		return "B+"
	case percentage >= 60:
		return "B"
	case percentage >= 50:
		return "C"
	case percentage >= 40:
		return "D"
	default:
		return "F"
	}
}
