package student

import (
	"context"
	"errors"
	"fmt"
	"github.com/ajaypatel01/CampusDesk/internal/platform/numtext"
	"strings"
	"time"

	"github.com/ajaypatel01/CampusDesk/internal/domain"
	apperr "github.com/ajaypatel01/CampusDesk/internal/platform/errors"
	"github.com/ajaypatel01/CampusDesk/internal/platform/httpx"
	"github.com/google/uuid"
)

// scholarNoEditRoles are the only roles allowed to change a student's scholar
// number (student_code) once set.
var scholarNoEditRoles = map[string]bool{
	"registrar":   true,
	"super_admin": true,
}

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
	EnrollmentNumber  string     `json:"enrollment_number"`
	AdmissionClass    string     `json:"admission_class"`
	AdmissionYear     string     `json:"admission_year"`
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
	EnrollmentNumber  string     `json:"enrollment_number"`
	AdmissionClass    string     `json:"admission_class"`
	AdmissionYear     string     `json:"admission_year"`
	PreviousSchool    string     `json:"previous_school"`
	BankName          string     `json:"bank_name"`
	BankIFSC          string     `json:"bank_ifsc"`
	BankAccountNumber string     `json:"bank_account_number"`
	BankHolderName    string     `json:"bank_holder_name"`
	BankBranch        string     `json:"bank_branch"`
	Status            string     `json:"status"`
	// TCDate/TCYear are required whenever Status is "inactive" -- see Update.
	TCDate *time.Time `json:"tc_date"`
	TCYear string     `json:"tc_year"`
}

var validSorts = map[string]bool{"name": true, "student_code": true, "admission_date": true, "class": true}
var validPaymentStatus = map[string]bool{"paid": true, "due": true, "partial": true, "unpaid": true}
var validGender = map[string]bool{"": true, "male": true, "female": true}

// normalizeGender lowercases and trims gender input, rejecting anything but male/female/blank.
func normalizeGender(g string) (string, error) {
	g = strings.ToLower(strings.TrimSpace(g))
	if !validGender[g] {
		return "", fmt.Errorf("%w: gender must be male or female", apperr.ErrInvalidInput)
	}
	return g, nil
}

func (s *Service) Create(ctx context.Context, in CreateInput) (*domain.Student, error) {
	if in.SchoolID == uuid.Nil || strings.TrimSpace(in.StudentCode) == "" ||
		strings.TrimSpace(in.FirstName) == "" {
		return nil, apperr.ErrInvalidInput
	}
	status := domain.StudentStatus(in.Status)
	if status == "" {
		status = domain.StudentStatusActive
	}
	gender, err := normalizeGender(in.Gender)
	if err != nil {
		return nil, err
	}
	st := &domain.Student{
		SchoolID:          in.SchoolID,
		StudentCode:       strings.TrimSpace(in.StudentCode),
		FirstName:         strings.TrimSpace(in.FirstName),
		LastName:          strings.TrimSpace(in.LastName),
		DateOfBirth:       in.DateOfBirth,
		Gender:            gender,
		Email:             strings.TrimSpace(in.Email),
		Phone:             numtext.Clean(in.Phone),
		Address:           strings.TrimSpace(in.Address),
		AdmissionDate:     in.AdmissionDate,
		Caste:             strings.TrimSpace(in.Caste),
		Category:          strings.TrimSpace(in.Category),
		AadharNumber:      numtext.Clean(in.AadharNumber),
		SamagraID:         numtext.Clean(in.SamagraID),
		PenNumber:         numtext.Clean(in.PenNumber),
		AparID:            numtext.Clean(in.AparID),
		EnrollmentNumber:  numtext.Clean(in.EnrollmentNumber),
		AdmissionClass:    strings.TrimSpace(in.AdmissionClass),
		AdmissionYear:     strings.TrimSpace(in.AdmissionYear),
		PreviousSchool:    strings.TrimSpace(in.PreviousSchool),
		BankName:          strings.TrimSpace(in.BankName),
		BankIFSC:          strings.TrimSpace(in.BankIFSC),
		BankAccountNumber: numtext.Clean(in.BankAccountNumber),
		BankHolderName:    strings.TrimSpace(in.BankHolderName),
		BankBranch:        strings.TrimSpace(in.BankBranch),
		Status:            status,
	}
	if err := s.repo.Create(ctx, st); err != nil {
		return nil, s.explainConflict(ctx, err, st.SchoolID, st.StudentCode, uuid.Nil)
	}
	return st, nil
}

// explainConflict turns the database's bare unique-violation on (school,
// scholar no.) into a message the person saving can act on. The only
// unique rule on students besides the id is that scholar no.
func (s *Service) explainConflict(ctx context.Context, err error, schoolID uuid.UUID, code string, selfID uuid.UUID) error {
	if !errors.Is(err, apperr.ErrConflict) {
		return err
	}
	name, lookupErr := s.repo.StudentNameByCode(ctx, schoolID, code, selfID)
	if lookupErr != nil || name == "" {
		return fmt.Errorf("%w: scholar no. %s is already used by another student in this school", apperr.ErrConflict, code)
	}
	return fmt.Errorf("%w: scholar no. %s is already used by %s -- each student in a school needs a different scholar no.", apperr.ErrConflict, code, name)
}

// ImportRowResult is one row's outcome from a bulk import -- returned for
// every row, success or failure, so the uploader gets a complete picture
// rather than an all-or-nothing result.
type ImportRowResult struct {
	RowNumber   int    `json:"row_number"`
	Success     bool   `json:"success"`
	StudentCode string `json:"student_code,omitempty"`
	Error       string `json:"error,omitempty"`
}

// BulkImport saves each parsed row through the exact same Create path as
// the single "Add Student" form -- a bulk-imported row is subject to the
// same validation as a manually-entered one, nothing looser. A row that
// fails (a bad value, a duplicate Student Code, ...) doesn't stop the rest
// of the batch; its outcome is just reported alongside the successful ones.
func (s *Service) BulkImport(ctx context.Context, schoolID uuid.UUID, rows []ImportRow) []ImportRowResult {
	results := make([]ImportRowResult, 0, len(rows))
	for _, row := range rows {
		if row.ParseErr != "" {
			results = append(results, ImportRowResult{RowNumber: row.RowNumber, Error: row.ParseErr})
			continue
		}
		in := row.Input
		in.SchoolID = schoolID
		st, err := s.Create(ctx, in)
		if err != nil {
			msg := err.Error()
			if apperr.IsConflict(err) {
				msg = fmt.Sprintf("a student with Student Code %q already exists at this school", in.StudentCode)
			}
			results = append(results, ImportRowResult{RowNumber: row.RowNumber, Error: msg})
			continue
		}
		results = append(results, ImportRowResult{RowNumber: row.RowNumber, Success: true, StudentCode: st.StudentCode})
	}
	return results
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
	if f.PaymentStatus != "" && !validPaymentStatus[f.PaymentStatus] {
		f.PaymentStatus = ""
	}
	return s.repo.List(ctx, f, limit, offset)
}

func (s *Service) Update(ctx context.Context, id uuid.UUID, in UpdateInput) (*domain.Student, error) {
	st, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	newCode := strings.TrimSpace(in.StudentCode)
	if newCode == "" || strings.TrimSpace(in.FirstName) == "" {
		return nil, apperr.ErrInvalidInput
	}
	if newCode != st.StudentCode {
		claims := httpx.ClaimsFromContext(ctx)
		if claims == nil || !scholarNoEditRoles[claims.Role] {
			return nil, apperr.ErrForbidden
		}
	}
	gender, err := normalizeGender(in.Gender)
	if err != nil {
		return nil, err
	}
	st.StudentCode = newCode
	st.FirstName = strings.TrimSpace(in.FirstName)
	st.LastName = strings.TrimSpace(in.LastName)
	st.DateOfBirth = in.DateOfBirth
	st.Gender = gender
	st.Email = strings.TrimSpace(in.Email)
	st.Phone = numtext.Clean(in.Phone)
	st.Address = strings.TrimSpace(in.Address)
	st.AdmissionDate = in.AdmissionDate
	st.Caste = strings.TrimSpace(in.Caste)
	st.Category = strings.TrimSpace(in.Category)
	st.AadharNumber = numtext.Clean(in.AadharNumber)
	st.SamagraID = numtext.Clean(in.SamagraID)
	st.PenNumber = numtext.Clean(in.PenNumber)
	st.AparID = numtext.Clean(in.AparID)
	st.EnrollmentNumber = numtext.Clean(in.EnrollmentNumber)
	st.AdmissionClass = strings.TrimSpace(in.AdmissionClass)
	st.AdmissionYear = strings.TrimSpace(in.AdmissionYear)
	st.PreviousSchool = strings.TrimSpace(in.PreviousSchool)
	st.BankName = strings.TrimSpace(in.BankName)
	st.BankIFSC = strings.TrimSpace(in.BankIFSC)
	st.BankAccountNumber = numtext.Clean(in.BankAccountNumber)
	st.BankHolderName = strings.TrimSpace(in.BankHolderName)
	st.BankBranch = strings.TrimSpace(in.BankBranch)
	prevStatus := st.Status
	if in.Status != "" {
		st.Status = domain.StudentStatus(in.Status)
	}
	// A duplicate drops out of every fee total, so it may not carry real
	// payments: those must first be moved to the student's correct record.
	if st.Status == domain.StudentStatusDuplicate && prevStatus != domain.StudentStatusDuplicate {
		paid, err := s.repo.PaymentsTotal(ctx, st.ID)
		if err != nil {
			return nil, err
		}
		if paid > 0 {
			return nil, fmt.Errorf("%w: this student has ₹%s in fee payments. Move them to the correct student's record (Fees → payment → Move) before marking this one as a duplicate", apperr.ErrInvalidInput, formatRupees(paid))
		}
	}
	// Marking a student inactive is what a Transfer Certificate is actually
	// issued for -- require both fields whenever that's the resulting
	// status, not just on the transition into it, so they can't be dropped
	// by a later edit either.
	if st.Status == domain.StudentStatusInactive {
		if in.TCDate == nil || strings.TrimSpace(in.TCYear) == "" {
			return nil, fmt.Errorf("%w: TC date and year are required to mark a student inactive", apperr.ErrInvalidInput)
		}
	}
	st.TCDate = in.TCDate
	st.TCYear = strings.TrimSpace(in.TCYear)
	if err := s.repo.Update(ctx, st); err != nil {
		return nil, s.explainConflict(ctx, err, st.SchoolID, st.StudentCode, st.ID)
	}
	return s.repo.GetByID(ctx, id)
}

func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}

// formatRupees writes 1234567 as "12,34,567" (Indian grouping).
func formatRupees(v int64) string {
	s := fmt.Sprintf("%d", v)
	if len(s) <= 3 {
		return s
	}
	head, tail := s[:len(s)-3], s[len(s)-3:]
	for len(head) > 2 {
		tail = head[len(head)-2:] + "," + tail
		head = head[:len(head)-2]
	}
	return head + "," + tail
}
