package payroll

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ajaypatel01/CampusDesk/internal/domain"
	"github.com/ajaypatel01/CampusDesk/internal/modules/school"
	"github.com/ajaypatel01/CampusDesk/internal/modules/staff"
	apperr "github.com/ajaypatel01/CampusDesk/internal/platform/errors"
	"github.com/google/uuid"
)

type Service struct {
	repo       *Repository
	staffRepo  *staff.Repository
	schoolRepo *school.Repository
}

func NewService(repo *Repository, staffRepo *staff.Repository, schoolRepo *school.Repository) *Service {
	return &Service{repo: repo, staffRepo: staffRepo, schoolRepo: schoolRepo}
}

var validLeaveTypes = map[string]bool{"cl": true, "unpaid": true}

type LeaveInput struct {
	UserID         uuid.UUID `json:"user_id"`
	AcademicYearID uuid.UUID `json:"academic_year_id"`
	LeaveType      string    `json:"leave_type"`
	StartDate      string    `json:"start_date"` // YYYY-MM-DD
	EndDate        string    `json:"end_date"`
	HalfDay        bool      `json:"half_day"` // only valid when start_date == end_date
	Reason         string    `json:"reason"`
}

func (s *Service) CreateLeave(ctx context.Context, in LeaveInput) (*domain.StaffLeave, error) {
	l, err := s.parseLeave(in)
	if err != nil {
		return nil, err
	}
	if err := s.repo.CreateLeave(ctx, l); err != nil {
		return nil, err
	}
	return l, nil
}

func (s *Service) UpdateLeave(ctx context.Context, id uuid.UUID, in LeaveInput) (*domain.StaffLeave, error) {
	l, err := s.parseLeave(in)
	if err != nil {
		return nil, err
	}
	l.ID = id
	if err := s.repo.UpdateLeave(ctx, l); err != nil {
		return nil, err
	}
	return s.repo.GetLeave(ctx, id)
}

func (s *Service) parseLeave(in LeaveInput) (*domain.StaffLeave, error) {
	leaveType := strings.ToLower(strings.TrimSpace(in.LeaveType))
	if leaveType == "" {
		leaveType = "cl"
	}
	if in.UserID == uuid.Nil || in.AcademicYearID == uuid.Nil || !validLeaveTypes[leaveType] {
		return nil, fmt.Errorf("%w: user_id, academic_year_id and a valid leave_type (cl/unpaid) are required", apperr.ErrInvalidInput)
	}
	start, err := time.Parse("2006-01-02", in.StartDate)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid start_date", apperr.ErrInvalidInput)
	}
	end, err := time.Parse("2006-01-02", in.EndDate)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid end_date", apperr.ErrInvalidInput)
	}
	if end.Before(start) {
		return nil, fmt.Errorf("%w: end_date must not be before start_date", apperr.ErrInvalidInput)
	}
	if in.HalfDay && !start.Equal(end) {
		return nil, fmt.Errorf("%w: half_day is only valid for a single-day leave (start_date must equal end_date)", apperr.ErrInvalidInput)
	}
	return &domain.StaffLeave{
		UserID:         in.UserID,
		AcademicYearID: in.AcademicYearID,
		LeaveType:      leaveType,
		StartDate:      start,
		EndDate:        end,
		HalfDay:        in.HalfDay,
		Reason:         strings.TrimSpace(in.Reason),
	}, nil
}

func (s *Service) DeleteLeave(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteLeave(ctx, id)
}

func (s *Service) ListLeaves(ctx context.Context, userID, academicYearID uuid.UUID) ([]domain.StaffLeave, error) {
	return s.repo.ListLeavesByUserYear(ctx, userID, academicYearID)
}

func (s *Service) ListLeavesForSchool(ctx context.Context, schoolID, academicYearID uuid.UUID) ([]domain.StaffLeave, error) {
	return s.repo.ListLeavesBySchoolYear(ctx, schoolID, academicYearID)
}

// MonthRow is one staff member's computed payroll for a single calendar month.
type MonthRow struct {
	UserID      uuid.UUID `json:"user_id"`
	FirstName   string    `json:"first_name"`
	LastName    string    `json:"last_name"`
	Designation string    `json:"designation,omitempty"`

	// Earnings breakdown; MonthlySalary (gross) = sum of the four below.
	BasicSalary      int `json:"basic_salary"`
	HRA              int `json:"hra"`
	SpecialAllowance int `json:"special_allowance"`
	Bonus            int `json:"bonus"`
	MonthlySalary    int `json:"monthly_salary"`

	WorkingDays  int     `json:"working_days_per_month"`
	PerDayRate   float64 `json:"per_day_rate"`
	CLPaidDays   float64 `json:"cl_paid_days"`
	CLExcessDays float64 `json:"cl_excess_days"`
	UnpaidDays   float64 `json:"unpaid_days"`
	DeductedDays float64 `json:"deducted_days"`

	// Deduction breakdown; Deduction (total, subtracted to get NetSalary) =
	// AttendanceDeduction + EPF + ESIC + AdditionalDeduction.
	AttendanceDeduction      int    `json:"attendance_deduction"`
	EPF                      int    `json:"epf"`
	ESIC                     int    `json:"esic"`
	AdditionalDeduction      int    `json:"additional_deduction"`
	AdditionalDeductionLabel string `json:"additional_deduction_label,omitempty"`
	Deduction                int    `json:"deduction"`
	NetSalary                int    `json:"net_salary"`

	CLQuota   int     `json:"cl_quota_per_year"`
	CLUsedYTD float64 `json:"cl_used_ytd"`
	CLBalance float64 `json:"cl_balance"`
}

// ComputeMonth computes every staff member's payroll for schoolID for the given
// calendar month, scoped to academicYearID for CL-quota tracking. A staff member is
// assumed present every day unless a staff_leaves record covers that date: "cl" leave
// draws down the year's CL quota (chronologically, oldest leave first) and only turns
// into a paid deduction once that quota is exhausted for the year; "unpaid" leave
// always deducts.
func (s *Service) ComputeMonth(ctx context.Context, schoolID, academicYearID uuid.UUID, year int, month time.Month) ([]MonthRow, error) {
	sch, err := s.schoolRepo.GetByID(ctx, schoolID)
	if err != nil {
		return nil, err
	}
	workingDays := sch.WorkingDaysPerMonth
	if workingDays <= 0 {
		workingDays = 30
	}

	members, _, err := s.staffRepo.List(ctx, &schoolID, 1000, 0)
	if err != nil {
		return nil, err
	}
	leaves, err := s.repo.ListLeavesBySchoolYear(ctx, schoolID, academicYearID)
	if err != nil {
		return nil, err
	}
	byUser := map[uuid.UUID][]domain.StaffLeave{}
	for _, l := range leaves {
		byUser[l.UserID] = append(byUser[l.UserID], l)
	}

	monthStart := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
	monthEnd := monthStart.AddDate(0, 1, -1)

	var rows []MonthRow
	for _, m := range members {
		if row, ok := computeMemberRow(m, byUser[m.ID], workingDays, monthStart, monthEnd); ok {
			rows = append(rows, row)
		}
	}
	return rows, nil
}

// computeMemberRow is the pure per-member half of ComputeMonth's calculation,
// shared with ComputeForUser so a single staff member's payroll can be computed
// without pulling (or exposing) the rest of the school's payroll.
func computeMemberRow(m domain.StaffMember, userLeaves []domain.StaffLeave, workingDays int, monthStart, monthEnd time.Time) (MonthRow, bool) {
	salary := 0
	quota := 7
	designation := ""
	basic, hra, specialAllowance, bonus := 0, 0, 0, 0
	epf, esic, additionalDeduction := 0, 0, 0
	additionalDeductionLabel := ""
	if m.Profile != nil {
		salary = m.Profile.Salary
		if m.Profile.CLQuotaPerYear > 0 {
			quota = m.Profile.CLQuotaPerYear
		}
		if m.Profile.Designation != nil {
			designation = *m.Profile.Designation
		}
		basic, hra, specialAllowance, bonus = m.Profile.BasicSalary, m.Profile.HRA, m.Profile.SpecialAllowance, m.Profile.Bonus
		epf, esic, additionalDeduction = m.Profile.EPF, m.Profile.ESIC, m.Profile.AdditionalDeduction
		if m.Profile.AdditionalDeductionLabel != nil {
			additionalDeductionLabel = *m.Profile.AdditionalDeductionLabel
		}
	}
	if salary <= 0 {
		return MonthRow{}, false // nothing to compute for staff with no salary on file
	}

	clUsedBeforeThisMonth := 0.0
	clDaysThisMonth := 0.0
	unpaidDaysThisMonth := 0.0
	for _, lv := range userLeaves {
		factor := 1.0
		if lv.HalfDay {
			factor = 0.5
		}
		if lv.LeaveType == "cl" && lv.EndDate.Before(monthStart) {
			clUsedBeforeThisMonth += factor * float64(daysBetween(lv.StartDate, lv.EndDate))
			continue
		}
		overlapDays, beforeMonthDays := overlap(lv.StartDate, lv.EndDate, monthStart, monthEnd)
		if lv.LeaveType == "cl" {
			clUsedBeforeThisMonth += factor * float64(beforeMonthDays)
			clDaysThisMonth += factor * float64(overlapDays)
		} else {
			unpaidDaysThisMonth += factor * float64(overlapDays)
		}
	}

	remainingBefore := float64(quota) - clUsedBeforeThisMonth
	if remainingBefore < 0 {
		remainingBefore = 0
	}
	paidCL := clDaysThisMonth
	if paidCL > remainingBefore {
		paidCL = remainingBefore
	}
	excessCL := clDaysThisMonth - paidCL

	deductedDays := excessCL + unpaidDaysThisMonth
	perDayRate := float64(salary) / float64(workingDays)
	attendanceDeduction := int(perDayRate*deductedDays + 0.5)
	if attendanceDeduction > salary {
		attendanceDeduction = salary
	}
	totalDeduction := attendanceDeduction + epf + esic + additionalDeduction
	if totalDeduction > salary {
		totalDeduction = salary
	}

	clUsedYTD := clUsedBeforeThisMonth + paidCL + excessCL
	clBalance := float64(quota) - clUsedYTD
	if clBalance < 0 {
		clBalance = 0
	}

	return MonthRow{
		UserID:                   m.ID,
		FirstName:                m.FirstName,
		LastName:                 m.LastName,
		Designation:              designation,
		BasicSalary:              basic,
		HRA:                      hra,
		SpecialAllowance:         specialAllowance,
		Bonus:                    bonus,
		MonthlySalary:            salary,
		WorkingDays:              workingDays,
		PerDayRate:               perDayRate,
		CLPaidDays:               paidCL,
		CLExcessDays:             excessCL,
		UnpaidDays:               unpaidDaysThisMonth,
		DeductedDays:             deductedDays,
		AttendanceDeduction:      attendanceDeduction,
		EPF:                      epf,
		ESIC:                     esic,
		AdditionalDeduction:      additionalDeduction,
		AdditionalDeductionLabel: additionalDeductionLabel,
		Deduction:                totalDeduction,
		NetSalary:                salary - totalDeduction,
		CLQuota:                  quota,
		CLUsedYTD:                clUsedYTD,
		CLBalance:                clBalance,
	}, true
}

// ComputeForUser computes a single staff member's payroll for the month without
// pulling the rest of the school's staff/leave data - used when a non-admin
// requests only their own salary.
func (s *Service) ComputeForUser(ctx context.Context, schoolID, academicYearID, userID uuid.UUID, year int, month time.Month) (*MonthRow, error) {
	sch, err := s.schoolRepo.GetByID(ctx, schoolID)
	if err != nil {
		return nil, err
	}
	workingDays := sch.WorkingDaysPerMonth
	if workingDays <= 0 {
		workingDays = 30
	}

	member, err := s.staffRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	leaves, err := s.repo.ListLeavesByUserYear(ctx, userID, academicYearID)
	if err != nil {
		return nil, err
	}

	monthStart := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
	monthEnd := monthStart.AddDate(0, 1, -1)

	row, ok := computeMemberRow(*member, leaves, workingDays, monthStart, monthEnd)
	if !ok {
		return nil, fmt.Errorf("%w: no salary on file for this staff member", apperr.ErrNotFound)
	}
	return &row, nil
}

// slipUnlockDate is the earliest a given month's salary slip may be downloaded:
// the 16th of the following month, i.e. strictly after the 15th, giving payroll
// time to finalize attendance/leave for the month before slips go out.
func slipUnlockDate(year int, month time.Month) time.Time {
	return time.Date(year, month, 16, 0, 0, 0, 0, time.UTC).AddDate(0, 1, 0)
}

// GenerateSlip computes one staff member's salary for the given month and renders
// it as a downloadable PDF, using the same present-day/CL logic as ComputeMonth.
func (s *Service) GenerateSlip(ctx context.Context, schoolID, academicYearID, userID uuid.UUID, year int, month time.Month) ([]byte, string, error) {
	if unlock := slipUnlockDate(year, month); time.Now().Before(unlock) {
		return nil, "", fmt.Errorf("%w: %s %d's salary slip is available from %s", apperr.ErrForbidden, month, year, unlock.Format("Jan 2, 2006"))
	}
	row, err := s.ComputeForUser(ctx, schoolID, academicYearID, userID, year, month)
	if err != nil {
		return nil, "", err
	}

	member, err := s.staffRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, "", err
	}
	sch, err := s.schoolRepo.GetByID(ctx, schoolID)
	if err != nil {
		return nil, "", err
	}

	data := SlipData{
		SchoolName:    sch.Name,
		SchoolAddress: sch.Address,
		SchoolPhone:   sch.Phone,
		SchoolEmail:   sch.Email,
		Logo:          schoolLogo(sch.Name),
		Signature:     authorizedSignature,
		Month:         month,
		Year:          year,
		Row:           *row,
	}
	if member.Profile != nil {
		data.BankName = derefStr(member.Profile.BankName)
		data.BankAccountNumber = derefStr(member.Profile.BankAccountNumber)
		data.BankIFSC = derefStr(member.Profile.BankIFSC)
	}

	pdfBytes, err := generateSalarySlipPDF(data)
	if err != nil {
		return nil, "", fmt.Errorf("generate salary slip: %w", err)
	}
	filename := fmt.Sprintf("salary_slip_%s_%d_%02d.pdf", userID.String()[:8], year, int(month))
	return pdfBytes, filename, nil
}

func derefStr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func daysBetween(start, end time.Time) int {
	return int(end.Sub(start).Hours()/24) + 1
}

// overlap returns how many days of [start,end] fall within [monthStart,monthEnd]
// (inclusive), and how many days of [start,end] fall strictly before monthStart.
func overlap(start, end, monthStart, monthEnd time.Time) (inMonth int, beforeMonth int) {
	if end.Before(monthStart) || start.After(monthEnd) {
		if end.Before(monthStart) {
			return 0, daysBetween(start, end)
		}
		return 0, 0
	}
	overlapStart := start
	if overlapStart.Before(monthStart) {
		beforeMonth = daysBetween(start, monthStart.AddDate(0, 0, -1))
		overlapStart = monthStart
	}
	overlapEnd := end
	if overlapEnd.After(monthEnd) {
		overlapEnd = monthEnd
	}
	inMonth = daysBetween(overlapStart, overlapEnd)
	return inMonth, beforeMonth
}
