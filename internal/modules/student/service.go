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

type StudentListItem struct {
	domain.Student
	GradeLevelName string `json:"grade_level_name,omitempty"`
	TotalDue       *int   `json:"total_due,omitempty"`
	TotalPaid      *int   `json:"total_paid,omitempty"`
	PendingFees    *int   `json:"pending_fees,omitempty"`
	FeeRemarks     string `json:"fee_remarks,omitempty"`
}

type CreateInput struct {
	SchoolID          uuid.UUID  `json:"school_id"`
	StudentCode       string     `json:"student_code"`
	FirstName         string     `json:"first_name"`
	LastName          string     `json:"last_name"`
	DateOfBirth       *time.Time `json:"date_of_birth"`
	Gender            string     `json:"gender"`
	Email             string     `json:"email"`
	Phone             string     `json:"phone"`
	Address           string     `json:"address"`
	AdmissionDate     *time.Time `json:"admission_date"`
	Caste             string     `json:"caste"`
	Category          string     `json:"category"`
	AadharNumber      string     `json:"aadhar_number"`
	SamagraID         string     `json:"samagra_id"`
	PenNumber         string     `json:"pen_number"`
	AparID            string     `json:"apar_id"`
	PreviousSchool    string     `json:"previous_school"`
	BankName          string     `json:"bank_name"`
	BankIFSC          string     `json:"bank_ifsc"`
	BankAccountNumber string     `json:"bank_account_number"`
	BankHolderName    string     `json:"bank_holder_name"`
	BankBranch        string     `json:"bank_branch"`
	Status            string     `json:"status"`
}

type UpdateInput struct {
	StudentCode       string     `json:"student_code"`
	FirstName         string     `json:"first_name"`
	LastName          string     `json:"last_name"`
	DateOfBirth       *time.Time `json:"date_of_birth"`
	Gender            string     `json:"gender"`
	Email             string     `json:"email"`
	Phone             string     `json:"phone"`
	Address           string     `json:"address"`
	AdmissionDate     *time.Time `json:"admission_date"`
	Caste             string     `json:"caste"`
	Category          string     `json:"category"`
	AadharNumber      string     `json:"aadhar_number"`
	SamagraID         string     `json:"samagra_id"`
	PenNumber         string     `json:"pen_number"`
	AparID            string     `json:"apar_id"`
	PreviousSchool    string     `json:"previous_school"`
	BankName          string     `json:"bank_name"`
	BankIFSC          string     `json:"bank_ifsc"`
	BankAccountNumber string     `json:"bank_account_number"`
	BankHolderName    string     `json:"bank_holder_name"`
	BankBranch        string     `json:"bank_branch"`
	Status            string     `json:"status"`
}

var validSorts = map[string]bool{"name": true, "student_code": true, "admission_date": true, "class": true}

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
		SchoolID:          in.SchoolID,
		StudentCode:       strings.TrimSpace(in.StudentCode),
		FirstName:         strings.TrimSpace(in.FirstName),
		LastName:          strings.TrimSpace(in.LastName),
		DateOfBirth:       in.DateOfBirth,
		Gender:            strings.TrimSpace(in.Gender),
		Email:             strings.TrimSpace(in.Email),
		Phone:             strings.TrimSpace(in.Phone),
		Address:           strings.TrimSpace(in.Address),
		AdmissionDate:     in.AdmissionDate,
		Caste:             strings.TrimSpace(in.Caste),
		Category:          strings.TrimSpace(in.Category),
		AadharNumber:      strings.TrimSpace(in.AadharNumber),
		SamagraID:         strings.TrimSpace(in.SamagraID),
		PenNumber:         strings.TrimSpace(in.PenNumber),
		AparID:            strings.TrimSpace(in.AparID),
		PreviousSchool:    strings.TrimSpace(in.PreviousSchool),
		BankName:          strings.TrimSpace(in.BankName),
		BankIFSC:          strings.TrimSpace(in.BankIFSC),
		BankAccountNumber: strings.TrimSpace(in.BankAccountNumber),
		BankHolderName:    strings.TrimSpace(in.BankHolderName),
		BankBranch:        strings.TrimSpace(in.BankBranch),
		Status:            status,
	}
	if err := s.repo.Create(ctx, st); err != nil {
		return nil, err
	}
	return st, nil
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (*domain.Student, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) List(ctx context.Context, f ListFilter, limit, offset int) ([]StudentListItem, int, error) {
	if f.SchoolID == uuid.Nil {
		return nil, 0, apperr.ErrInvalidInput
	}
	if f.SortOrder != "desc" {
		f.SortOrder = "asc"
	}
	if !validSorts[f.SortBy] {
		f.SortBy = "name"
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
	st.AdmissionDate = in.AdmissionDate
	st.Caste = strings.TrimSpace(in.Caste)
	st.Category = strings.TrimSpace(in.Category)
	st.AadharNumber = strings.TrimSpace(in.AadharNumber)
	st.SamagraID = strings.TrimSpace(in.SamagraID)
	st.PenNumber = strings.TrimSpace(in.PenNumber)
	st.AparID = strings.TrimSpace(in.AparID)
	st.PreviousSchool = strings.TrimSpace(in.PreviousSchool)
	st.BankName = strings.TrimSpace(in.BankName)
	st.BankIFSC = strings.TrimSpace(in.BankIFSC)
	st.BankAccountNumber = strings.TrimSpace(in.BankAccountNumber)
	st.BankHolderName = strings.TrimSpace(in.BankHolderName)
	st.BankBranch = strings.TrimSpace(in.BankBranch)
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
