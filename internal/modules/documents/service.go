package documents

import (
	"context"
	"fmt"
	"time"

	apperr "github.com/ajaypatel01/CampusDesk/internal/platform/errors"
	"github.com/google/uuid"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// ---- Data types ----

type BonafideData struct {
	SchoolName    string
	SchoolAddress string
	SchoolPhone   string
	SchoolEmail   string

	StudentName   string
	StudentCode   string
	DOB           *time.Time
	Gender        string
	Category      string
	Caste         string
	AdmissionDate *time.Time

	ClassName    string
	AcademicYear string

	GuardianName     string
	GuardianRelation string

	IssueDate time.Time
}

type TCData struct {
	SchoolName    string
	SchoolAddress string
	SchoolPhone   string
	SchoolEmail   string

	StudentName    string
	StudentCode    string
	DOB            *time.Time
	Gender         string
	Category       string
	Caste          string
	AdmissionDate  *time.Time
	AadharNumber   string
	PreviousSchool string

	AdmittedClass    string
	LastClass        string
	LastAcademicYear string

	GuardianName     string
	GuardianRelation string

	DateOfLeaving    time.Time
	ReasonForLeaving string
	Conduct          string
	FeeCleared       bool
	OutstandingFees  int

	IssueDate time.Time
}

type SalarySlipInput struct {
	SchoolID     uuid.UUID `json:"school_id"`
	EmployeeName string    `json:"employee_name"`
	EmployeeID   string    `json:"employee_id"`
	Designation  string    `json:"designation"`
	Department   string    `json:"department"`
	Month        string    `json:"month"`
	Year         int       `json:"year"`

	BasicSalary      int `json:"basic_salary"`
	HRA              int `json:"hra"`
	DA               int `json:"da"`
	TA               int `json:"ta"`
	MedicalAllowance int `json:"medical_allowance"`
	OtherAllowance   int `json:"other_allowance"`

	PF             int `json:"pf"`
	TDS            int `json:"tds"`
	ESI            int `json:"esi"`
	OtherDeduction int `json:"other_deduction"`
}

type SalarySlipData struct {
	SalarySlipInput

	SchoolName    string
	SchoolAddress string
	SchoolPhone   string
	SchoolEmail   string

	GrossSalary    int
	TotalDeduction int
	NetSalary      int
	IssueDate      time.Time
}

// ---- Service methods ----

func (s *Service) GenerateBonafide(ctx context.Context, studentID, yearID uuid.UUID) ([]byte, string, error) {
	if studentID == uuid.Nil || yearID == uuid.Nil {
		return nil, "", apperr.ErrInvalidInput
	}
	data, err := s.repo.GetBonafideData(ctx, studentID, yearID)
	if err != nil {
		return nil, "", err
	}
	pdf, err := generateBonafidePDF(*data)
	if err != nil {
		return nil, "", fmt.Errorf("generate bonafide: %w", err)
	}
	filename := fmt.Sprintf("bonafide_%s.pdf", studentID.String()[:8])
	return pdf, filename, nil
}

func (s *Service) GenerateTC(ctx context.Context, studentID uuid.UUID, dateOfLeaving time.Time, reason, conduct string) ([]byte, string, error) {
	if studentID == uuid.Nil {
		return nil, "", apperr.ErrInvalidInput
	}
	data, err := s.repo.GetTCData(ctx, studentID)
	if err != nil {
		return nil, "", err
	}
	data.DateOfLeaving = dateOfLeaving
	if data.DateOfLeaving.IsZero() {
		data.DateOfLeaving = time.Now()
	}
	data.ReasonForLeaving = reason
	if data.ReasonForLeaving == "" {
		data.ReasonForLeaving = "Transfer"
	}
	data.Conduct = conduct
	if data.Conduct == "" {
		data.Conduct = "Good"
	}
	pdf, err := generateTCPDF(*data)
	if err != nil {
		return nil, "", fmt.Errorf("generate tc: %w", err)
	}
	filename := fmt.Sprintf("tc_%s.pdf", studentID.String()[:8])
	return pdf, filename, nil
}

func (s *Service) GenerateSalarySlip(ctx context.Context, in SalarySlipInput) ([]byte, string, error) {
	if in.EmployeeName == "" || in.Month == "" || in.Year == 0 {
		return nil, "", apperr.ErrInvalidInput
	}
	data := SalarySlipData{
		SalarySlipInput: in,
		IssueDate:       time.Now(),
	}
	if in.SchoolID != uuid.Nil {
		name, addr, phone, email, err := s.repo.GetSchoolName(ctx, in.SchoolID)
		if err == nil {
			data.SchoolName = name
			data.SchoolAddress = addr
			data.SchoolPhone = phone
			data.SchoolEmail = email
		}
	}
	data.GrossSalary = in.BasicSalary + in.HRA + in.DA + in.TA + in.MedicalAllowance + in.OtherAllowance
	data.TotalDeduction = in.PF + in.TDS + in.ESI + in.OtherDeduction
	data.NetSalary = data.GrossSalary - data.TotalDeduction

	pdf, err := generateSalarySlipPDF(data)
	if err != nil {
		return nil, "", fmt.Errorf("generate salary slip: %w", err)
	}
	filename := fmt.Sprintf("salary_%s_%d.pdf", in.Month, in.Year)
	if in.EmployeeID != "" {
		filename = fmt.Sprintf("salary_%s_%s_%d.pdf", in.EmployeeID, in.Month, in.Year)
	}
	return pdf, filename, nil
}
