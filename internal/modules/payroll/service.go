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
	return &domain.StaffLeave{
		UserID:         in.UserID,
		AcademicYearID: in.AcademicYearID,
		LeaveType:      leaveType,
		StartDate:      start,
		EndDate:        end,
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
	UserID        uuid.UUID `json:"user_id"`
	FirstName     string    `json:"first_name"`
	LastName      string    `json:"last_name"`
	Designation   string    `json:"designation,omitempty"`
	MonthlySalary int       `json:"monthly_salary"`
	WorkingDays   int       `json:"working_days_per_month"`
	PerDayRate    float64   `json:"per_day_rate"`
	CLPaidDays    int       `json:"cl_paid_days"`
	CLExcessDays  int       `json:"cl_excess_days"`
	UnpaidDays    int       `json:"unpaid_days"`
	DeductedDays  int       `json:"deducted_days"`
	Deduction     int       `json:"deduction"`
	NetSalary     int       `json:"net_salary"`
	CLQuota       int       `json:"cl_quota_per_year"`
	CLUsedYTD     int       `json:"cl_used_ytd"`
	CLBalance     int       `json:"cl_balance"`
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
		workingDays = 26
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
		salary := 0
		quota := 12
		designation := ""
		if m.Profile != nil {
			salary = m.Profile.Salary
			if m.Profile.CLQuotaPerYear > 0 {
				quota = m.Profile.CLQuotaPerYear
			}
			if m.Profile.Designation != nil {
				designation = *m.Profile.Designation
			}
		}
		if salary <= 0 {
			continue // nothing to compute for staff with no salary on file
		}

		userLeaves := byUser[m.ID]
		clUsedBeforeThisMonth := 0
		clDaysThisMonth := 0
		unpaidDaysThisMonth := 0
		for _, lv := range userLeaves {
			if lv.LeaveType == "cl" && lv.EndDate.Before(monthStart) {
				clUsedBeforeThisMonth += daysBetween(lv.StartDate, lv.EndDate)
				continue
			}
			overlapDays, beforeMonthDays := overlap(lv.StartDate, lv.EndDate, monthStart, monthEnd)
			if lv.LeaveType == "cl" {
				clUsedBeforeThisMonth += beforeMonthDays
				clDaysThisMonth += overlapDays
			} else {
				unpaidDaysThisMonth += overlapDays
			}
		}

		remainingBefore := quota - clUsedBeforeThisMonth
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
		deduction := int(perDayRate*float64(deductedDays) + 0.5)
		if deduction > salary {
			deduction = salary
		}

		clUsedYTD := clUsedBeforeThisMonth + paidCL + excessCL
		clBalance := quota - clUsedYTD
		if clBalance < 0 {
			clBalance = 0
		}

		rows = append(rows, MonthRow{
			UserID:        m.ID,
			FirstName:     m.FirstName,
			LastName:      m.LastName,
			Designation:   designation,
			MonthlySalary: salary,
			WorkingDays:   workingDays,
			PerDayRate:    perDayRate,
			CLPaidDays:    paidCL,
			CLExcessDays:  excessCL,
			UnpaidDays:    unpaidDaysThisMonth,
			DeductedDays:  deductedDays,
			Deduction:     deduction,
			NetSalary:     salary - deduction,
			CLQuota:       quota,
			CLUsedYTD:     clUsedYTD,
			CLBalance:     clBalance,
		})
	}
	return rows, nil
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
