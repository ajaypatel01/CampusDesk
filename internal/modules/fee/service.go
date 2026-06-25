package fee

import (
	"context"
	"fmt"
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

// ---- Input types ----

type CreateFeeStructureInput struct {
	SchoolID         uuid.UUID                    `json:"school_id"`
	AcademicYearID   uuid.UUID                    `json:"academic_year_id"`
	GradeLevelID     uuid.UUID                    `json:"grade_level_id"`
	TuitionFeeAnnual int                          `json:"tuition_fee_annual"`
	NumInstallments  int                          `json:"num_installments"`
	VanFeeAnnual     int                          `json:"van_fee_annual"`
	Installments     []CreateInstallmentPlanInput `json:"installments"`
}

type CreateInstallmentPlanInput struct {
	InstallmentNumber int       `json:"installment_number"`
	Label             string    `json:"label"`
	Amount            int       `json:"amount"`
	DueDate           time.Time `json:"due_date"`
}

type UpdateFeeStructureInput struct {
	TuitionFeeAnnual int                          `json:"tuition_fee_annual"`
	NumInstallments  int                          `json:"num_installments"`
	VanFeeAnnual     int                          `json:"van_fee_annual"`
	Installments     []CreateInstallmentPlanInput `json:"installments"`
}

type CreateFeeAccountInput struct {
	StudentID        uuid.UUID `json:"student_id"`
	SchoolID         uuid.UUID `json:"school_id"`
	AcademicYearID   uuid.UUID `json:"academic_year_id"`
	FeeStructureID   uuid.UUID `json:"fee_structure_id"`
	TuitionFee       int       `json:"tuition_fee"`
	DiscountAmount   int       `json:"discount_amount"`
	DiscountReason   string    `json:"discount_reason"`
	PreviousYearDues int       `json:"previous_year_dues"`
	VanFee           int       `json:"van_fee"`
	IsRTE            bool      `json:"is_rte"`
}

type UpdateFeeAccountInput struct {
	DiscountAmount   *int    `json:"discount_amount"`
	DiscountReason   *string `json:"discount_reason"`
	PreviousYearDues *int    `json:"previous_year_dues"`
	VanFee           *int    `json:"van_fee"`
	IsRTE            *bool   `json:"is_rte"`
}

type RecordPaymentInput struct {
	StudentFeeAccountID uuid.UUID `json:"student_fee_account_id"`
	FeeType             string    `json:"fee_type"`
	InstallmentNumber   *int      `json:"installment_number"`
	Amount              int       `json:"amount"`
	PaymentDate         time.Time `json:"payment_date"`
	PaymentMode         string    `json:"payment_mode"`
	ReferenceNumber     string    `json:"reference_number"`
	Notes               string    `json:"notes"`
}

// ---- Response types ----

type FeeStructureDetail struct {
	domain.FeeStructure
	Installments []domain.FeeInstallmentPlan `json:"installments"`
}

type FeeAccountDetail struct {
	domain.StudentFeeAccount
	StudentName      string              `json:"student_name"`
	StudentCode      string              `json:"student_code"`
	GradeLevelName   string              `json:"grade_level_name"`
	Payments         []domain.FeePayment `json:"payments"`
	TotalPaid        int                 `json:"total_paid"`
	TuitionPaid      int                 `json:"tuition_paid"`
	VanPaid          int                 `json:"van_paid"`
	PreviousDuesPaid int                 `json:"previous_dues_paid"`
	TotalDue         int                 `json:"total_due"`
	BalanceRemaining int                 `json:"balance_remaining"`
}

type FeeAccountSummary struct {
	ID               uuid.UUID `json:"id"`
	StudentID        uuid.UUID `json:"student_id"`
	StudentName      string    `json:"student_name"`
	StudentCode      string    `json:"student_code"`
	GradeLevelName   string    `json:"grade_level_name"`
	TuitionFee       int       `json:"tuition_fee"`
	DiscountAmount   int       `json:"discount_amount"`
	VanFee           int       `json:"van_fee"`
	PreviousYearDues int       `json:"previous_year_dues"`
	IsRTE            bool      `json:"is_rte"`
	TotalDue         int       `json:"total_due"`
	TotalPaid        int       `json:"total_paid"`
	BalanceRemaining int       `json:"balance_remaining"`
}

type FeeAccountFilter struct {
	SchoolID       uuid.UUID
	AcademicYearID uuid.UUID
	Search         string
	GradeLevel     string
	PaymentStatus  string // "paid", "due", "partial"
}

type SchoolFeeSummaryResponse struct {
	SchoolID         uuid.UUID         `json:"school_id"`
	AcademicYearID   uuid.UUID         `json:"academic_year_id"`
	TotalStudents    int               `json:"total_students"`
	RTEStudents      int               `json:"rte_students"`
	TotalTuitionDue  int               `json:"total_tuition_due"`
	TotalVanDue      int               `json:"total_van_due"`
	TotalPrevDue     int               `json:"total_prev_due"`
	TotalDiscount    int               `json:"total_discount"`
	GrandTotalDue    int               `json:"grand_total_due"`
	TotalCollected   int               `json:"total_collected"`
	TotalOutstanding int               `json:"total_outstanding"`
	ByGrade          []GradeFeeSummary `json:"by_grade"`
}

type GradeFeeSummary struct {
	GradeLevelID   uuid.UUID `json:"grade_level_id"`
	GradeLevelName string    `json:"grade_level_name"`
	StudentCount   int       `json:"student_count"`
	TotalDue       int       `json:"total_due"`
	TotalCollected int       `json:"total_collected"`
	Outstanding    int       `json:"outstanding"`
}

type StudentFeeSummaryResponse struct {
	StudentID        uuid.UUID           `json:"student_id"`
	StudentName      string              `json:"student_name"`
	StudentCode      string              `json:"student_code"`
	AcademicYearID   uuid.UUID           `json:"academic_year_id"`
	GradeLevelName   string              `json:"grade_level_name"`
	TuitionFee       int                 `json:"tuition_fee"`
	DiscountAmount   int                 `json:"discount_amount"`
	DiscountReason   string              `json:"discount_reason,omitempty"`
	NetTuitionFee    int                 `json:"net_tuition_fee"`
	VanFee           int                 `json:"van_fee"`
	PreviousYearDues int                 `json:"previous_year_dues"`
	IsRTE            bool                `json:"is_rte"`
	TotalDue         int                 `json:"total_due"`
	TotalPaid        int                 `json:"total_paid"`
	BalanceRemaining int                 `json:"balance_remaining"`
	Payments         []domain.FeePayment `json:"payments"`
}

type ReceiptData struct {
	PaymentID       uuid.UUID
	FeeType         string
	InstallmentNum  *int
	Amount          int
	PaymentDate     time.Time
	PaymentMode     string
	ReferenceNumber string
	Notes           string

	SchoolName    string
	SchoolAddress string
	SchoolPhone   string
	SchoolEmail   string

	StudentName    string
	StudentCode    string
	GradeLevelName string
	FatherName     string

	TuitionFee       int
	DiscountAmount   int
	VanFee           int
	PreviousYearDues int
	TotalDue         int
	TotalPaidOther   int
	TotalPaidAfter   int
	BalanceAfter     int

	AcademicYearName string
	FeeAccountID     uuid.UUID
}

// ---- Service methods ----

func (s *Service) CreateFeeStructure(ctx context.Context, in CreateFeeStructureInput) (*FeeStructureDetail, error) {
	if in.SchoolID == uuid.Nil || in.AcademicYearID == uuid.Nil || in.GradeLevelID == uuid.Nil {
		return nil, apperr.ErrInvalidInput
	}
	if in.TuitionFeeAnnual <= 0 {
		return nil, apperr.ErrInvalidInput
	}
	if in.NumInstallments <= 0 {
		in.NumInstallments = 4
	}

	fs := &domain.FeeStructure{
		SchoolID:         in.SchoolID,
		AcademicYearID:   in.AcademicYearID,
		GradeLevelID:     in.GradeLevelID,
		TuitionFeeAnnual: in.TuitionFeeAnnual,
		NumInstallments:  in.NumInstallments,
		VanFeeAnnual:     in.VanFeeAnnual,
	}

	var plans []domain.FeeInstallmentPlan
	if len(in.Installments) > 0 {
		for _, ip := range in.Installments {
			plans = append(plans, domain.FeeInstallmentPlan{
				InstallmentNumber: ip.InstallmentNumber,
				Label:             ip.Label,
				Amount:            ip.Amount,
				DueDate:           ip.DueDate,
			})
		}
	} else {
		perQ := in.TuitionFeeAnnual / in.NumInstallments
		labels := []string{"Q1 (Apr-Jun)", "Q2 (Jul-Sep)", "Q3 (Oct-Dec)", "Q4 (Jan-Mar)"}
		months := []time.Month{time.April, time.July, time.October, time.January}
		for i := 0; i < in.NumInstallments && i < 4; i++ {
			year := 2025
			if months[i] == time.January {
				year = 2026
			}
			amt := perQ
			if i == in.NumInstallments-1 {
				amt = in.TuitionFeeAnnual - perQ*(in.NumInstallments-1)
			}
			plans = append(plans, domain.FeeInstallmentPlan{
				InstallmentNumber: i + 1,
				Label:             labels[i],
				Amount:            amt,
				DueDate:           time.Date(year, months[i], 1, 0, 0, 0, 0, time.UTC),
			})
		}
	}

	if err := s.repo.CreateFeeStructure(ctx, fs, plans); err != nil {
		return nil, err
	}
	return &FeeStructureDetail{FeeStructure: *fs, Installments: plans}, nil
}

func (s *Service) GetFeeStructure(ctx context.Context, id uuid.UUID) (*FeeStructureDetail, error) {
	fs, err := s.repo.GetFeeStructureByID(ctx, id)
	if err != nil {
		return nil, err
	}
	plans, err := s.repo.ListInstallmentPlans(ctx, id)
	if err != nil {
		return nil, err
	}
	return &FeeStructureDetail{FeeStructure: *fs, Installments: plans}, nil
}

func (s *Service) ListFeeStructures(ctx context.Context, schoolID, yearID uuid.UUID) ([]domain.FeeStructure, error) {
	if schoolID == uuid.Nil || yearID == uuid.Nil {
		return nil, apperr.ErrInvalidInput
	}
	return s.repo.ListFeeStructures(ctx, schoolID, yearID)
}

func (s *Service) UpdateFeeStructure(ctx context.Context, id uuid.UUID, in UpdateFeeStructureInput) (*FeeStructureDetail, error) {
	fs, err := s.repo.GetFeeStructureByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if in.TuitionFeeAnnual > 0 {
		fs.TuitionFeeAnnual = in.TuitionFeeAnnual
	}
	if in.NumInstallments > 0 {
		fs.NumInstallments = in.NumInstallments
	}
	fs.VanFeeAnnual = in.VanFeeAnnual

	var plans []domain.FeeInstallmentPlan
	for _, ip := range in.Installments {
		plans = append(plans, domain.FeeInstallmentPlan{
			InstallmentNumber: ip.InstallmentNumber,
			Label:             ip.Label,
			Amount:            ip.Amount,
			DueDate:           ip.DueDate,
		})
	}

	if err := s.repo.UpdateFeeStructure(ctx, fs, plans); err != nil {
		return nil, err
	}
	return s.GetFeeStructure(ctx, id)
}

func (s *Service) CreateFeeAccount(ctx context.Context, in CreateFeeAccountInput) (*domain.StudentFeeAccount, error) {
	if in.StudentID == uuid.Nil || in.SchoolID == uuid.Nil || in.AcademicYearID == uuid.Nil || in.FeeStructureID == uuid.Nil {
		return nil, apperr.ErrInvalidInput
	}
	fa := &domain.StudentFeeAccount{
		StudentID:        in.StudentID,
		SchoolID:         in.SchoolID,
		AcademicYearID:   in.AcademicYearID,
		FeeStructureID:   in.FeeStructureID,
		TuitionFee:       in.TuitionFee,
		DiscountAmount:   in.DiscountAmount,
		DiscountReason:   in.DiscountReason,
		PreviousYearDues: in.PreviousYearDues,
		VanFee:           in.VanFee,
		IsRTE:            in.IsRTE,
	}
	if fa.IsRTE {
		fa.TuitionFee = 0
		fa.DiscountAmount = 0
	}
	if err := s.repo.CreateFeeAccount(ctx, fa); err != nil {
		return nil, err
	}
	return fa, nil
}

func (s *Service) GetFeeAccount(ctx context.Context, id uuid.UUID) (*FeeAccountDetail, error) {
	fa, err := s.repo.GetFeeAccountByID(ctx, id)
	if err != nil {
		return nil, err
	}
	payments, err := s.repo.ListPayments(ctx, id)
	if err != nil {
		return nil, err
	}

	detail := &FeeAccountDetail{
		StudentFeeAccount: *fa,
		Payments:          payments,
		TotalDue:          fa.TuitionFee - fa.DiscountAmount + fa.VanFee + fa.PreviousYearDues,
	}
	for _, p := range payments {
		if p.Voided {
			continue
		}
		detail.TotalPaid += p.Amount
		switch p.FeeType {
		case domain.FeeTypeTuition:
			detail.TuitionPaid += p.Amount
		case domain.FeeTypeVan:
			detail.VanPaid += p.Amount
		case domain.FeeTypePreviousDues:
			detail.PreviousDuesPaid += p.Amount
		}
	}
	detail.BalanceRemaining = detail.TotalDue - detail.TotalPaid

	name, code, grade, err := s.repo.GetStudentInfo(ctx, fa.StudentID)
	if err == nil {
		detail.StudentName = name
		detail.StudentCode = code
		detail.GradeLevelName = grade
	}

	return detail, nil
}

func (s *Service) ListFeeAccounts(ctx context.Context, f FeeAccountFilter, limit, offset int) ([]FeeAccountSummary, int, error) {
	if f.SchoolID == uuid.Nil || f.AcademicYearID == uuid.Nil {
		return nil, 0, apperr.ErrInvalidInput
	}
	return s.repo.ListFeeAccounts(ctx, f, limit, offset)
}

func (s *Service) UpdateFeeAccount(ctx context.Context, id uuid.UUID, in UpdateFeeAccountInput) (*domain.StudentFeeAccount, error) {
	fa, err := s.repo.GetFeeAccountByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if in.DiscountAmount != nil {
		fa.DiscountAmount = *in.DiscountAmount
	}
	if in.DiscountReason != nil {
		fa.DiscountReason = *in.DiscountReason
	}
	if in.PreviousYearDues != nil {
		fa.PreviousYearDues = *in.PreviousYearDues
	}
	if in.VanFee != nil {
		fa.VanFee = *in.VanFee
	}
	if in.IsRTE != nil {
		fa.IsRTE = *in.IsRTE
		if fa.IsRTE {
			fa.TuitionFee = 0
			fa.DiscountAmount = 0
		}
	}
	if err := s.repo.UpdateFeeAccount(ctx, fa); err != nil {
		return nil, err
	}
	return s.repo.GetFeeAccountByID(ctx, id)
}

func (s *Service) RecordPayment(ctx context.Context, in RecordPaymentInput) (*domain.FeePayment, error) {
	if in.StudentFeeAccountID == uuid.Nil || in.Amount <= 0 {
		return nil, apperr.ErrInvalidInput
	}
	ft := domain.FeeType(in.FeeType)
	if ft != domain.FeeTypeTuition && ft != domain.FeeTypeVan && ft != domain.FeeTypePreviousDues {
		return nil, apperr.ErrInvalidInput
	}
	pm := domain.PaymentMode(in.PaymentMode)
	if pm == "" {
		pm = domain.PaymentModeCash
	}
	if _, err := s.repo.GetFeeAccountByID(ctx, in.StudentFeeAccountID); err != nil {
		return nil, err
	}
	payDate := in.PaymentDate
	if payDate.IsZero() {
		payDate = time.Now().UTC()
	}
	p := &domain.FeePayment{
		StudentFeeAccountID: in.StudentFeeAccountID,
		FeeType:             ft,
		InstallmentNumber:   in.InstallmentNumber,
		Amount:              in.Amount,
		PaymentDate:         payDate,
		PaymentMode:         pm,
		ReferenceNumber:     in.ReferenceNumber,
		Notes:               in.Notes,
	}
	if err := s.repo.CreatePayment(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *Service) ListPayments(ctx context.Context, accountID uuid.UUID) ([]domain.FeePayment, error) {
	if accountID == uuid.Nil {
		return nil, apperr.ErrInvalidInput
	}
	return s.repo.ListPayments(ctx, accountID)
}

func (s *Service) VoidPayment(ctx context.Context, id uuid.UUID) error {
	return s.repo.VoidPayment(ctx, id)
}

func (s *Service) SchoolFeeSummary(ctx context.Context, schoolID, yearID uuid.UUID) (*SchoolFeeSummaryResponse, error) {
	if schoolID == uuid.Nil || yearID == uuid.Nil {
		return nil, apperr.ErrInvalidInput
	}
	return s.repo.SchoolFeeSummary(ctx, schoolID, yearID)
}

func (s *Service) GenerateReceipt(ctx context.Context, paymentID uuid.UUID) ([]byte, string, error) {
	data, err := s.repo.GetReceiptData(ctx, paymentID)
	if err != nil {
		return nil, "", err
	}
	pdfBytes, err := generateReceiptPDF(*data)
	if err != nil {
		return nil, "", fmt.Errorf("generate receipt: %w", err)
	}
	filename := fmt.Sprintf("receipt_%s.pdf", paymentID.String()[:8])
	return pdfBytes, filename, nil
}

func (s *Service) StudentFeeSummary(ctx context.Context, studentID, yearID uuid.UUID) (*StudentFeeSummaryResponse, error) {
	if studentID == uuid.Nil || yearID == uuid.Nil {
		return nil, apperr.ErrInvalidInput
	}
	fa, err := s.repo.GetFeeAccountByStudent(ctx, studentID, yearID)
	if err != nil {
		return nil, err
	}
	payments, err := s.repo.ListPayments(ctx, fa.ID)
	if err != nil {
		return nil, err
	}

	name, code, grade, _ := s.repo.GetStudentInfo(ctx, studentID)
	netTuition := fa.TuitionFee - fa.DiscountAmount
	totalDue := netTuition + fa.VanFee + fa.PreviousYearDues
	var totalPaid int
	for _, p := range payments {
		if !p.Voided {
			totalPaid += p.Amount
		}
	}

	return &StudentFeeSummaryResponse{
		StudentID:        studentID,
		StudentName:      name,
		StudentCode:      code,
		AcademicYearID:   yearID,
		GradeLevelName:   grade,
		TuitionFee:       fa.TuitionFee,
		DiscountAmount:   fa.DiscountAmount,
		DiscountReason:   fa.DiscountReason,
		NetTuitionFee:    netTuition,
		VanFee:           fa.VanFee,
		PreviousYearDues: fa.PreviousYearDues,
		IsRTE:            fa.IsRTE,
		TotalDue:         totalDue,
		TotalPaid:        totalPaid,
		BalanceRemaining: totalDue - totalPaid,
		Payments:         payments,
	}, nil
}
