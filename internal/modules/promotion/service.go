// Package promotion moves a student to a different grade (promotion,
// demotion, or a mid-year class change) and/or a different class section.
//
// This app's actual source of truth for "what grade is a student in, for a
// given academic year" is their student_fee_accounts row (see
// fee.Repository.GetFeeStructureByGrade's doc comment) -- not
// enrollments.class_section_id, which is left NULL for effectively every
// enrollment in production. So "promoting" a student is really: point their
// fee account for the target year at the target grade's fee structure. This
// package also keeps an enrollments row in sync (creating one if the
// student didn't have one for that year yet) so section assignment,
// broadcasts-by-grade, and homework-by-section -- which all join through
// enrollments -- have something real to join against going forward.
package promotion

import (
	"context"
	"fmt"
	"time"

	"github.com/ajaypatel01/CampusDesk/internal/domain"
	"github.com/ajaypatel01/CampusDesk/internal/modules/academic"
	"github.com/ajaypatel01/CampusDesk/internal/modules/enrollment"
	"github.com/ajaypatel01/CampusDesk/internal/modules/fee"
	apperr "github.com/ajaypatel01/CampusDesk/internal/platform/errors"
	"github.com/google/uuid"
)

// studentGetter is the one student.Repository method this package needs.
// Declared locally (rather than importing internal/modules/student) so the
// student module can hold a *promotion.Handler without creating an import
// cycle -- student.Repository already satisfies this interface as-is.
type studentGetter interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Student, error)
}

type Service struct {
	students studentGetter
	academic *academic.Repository
	fees     *fee.Repository
	enroll   *enrollment.Repository
}

func NewService(students studentGetter, academicRepo *academic.Repository, fees *fee.Repository, enroll *enrollment.Repository) *Service {
	return &Service{students: students, academic: academicRepo, fees: fees, enroll: enroll}
}

// MoveGradeInput describes moving one student to a (possibly new) grade in a
// (possibly new) academic year. FromAcademicYearID is the year to carry
// forward discount/van-fee/dues from -- leave it uuid.Nil for a mid-year
// class change where there's nothing to carry forward from, only a fee
// account to repoint at the new grade.
type MoveGradeInput struct {
	StudentID            uuid.UUID
	FromAcademicYearID   uuid.UUID
	ToAcademicYearID     uuid.UUID
	ToGradeLevelID       uuid.UUID
	ClassSectionID       *uuid.UUID
	CarryForwardDues     bool
	CarryForwardDiscount bool
	CarryForwardVanFee   bool
}

type MoveGradeResult struct {
	StudentID      uuid.UUID `json:"student_id"`
	AccountID      uuid.UUID `json:"account_id"`
	GradeLevelID   uuid.UUID `json:"grade_level_id"`
	GradeLevelName string    `json:"grade_level_name"`
}

// MoveGrade points a student's fee account for ToAcademicYearID at
// ToGradeLevelID's fee structure. If the student already has an account for
// that year (a same-year reassignment), it's updated in place; otherwise a
// new one is created, optionally carrying forward discount/van-fee/RTE flag
// and outstanding balance (as previous_year_dues) from FromAcademicYearID.
func (s *Service) MoveGrade(ctx context.Context, in MoveGradeInput) (*MoveGradeResult, error) {
	st, err := s.students.GetByID(ctx, in.StudentID)
	if err != nil {
		return nil, err
	}
	grade, err := s.academic.GetGrade(ctx, in.ToGradeLevelID)
	if err != nil {
		return nil, fmt.Errorf("%w: grade not found", apperr.ErrInvalidInput)
	}
	if grade.SchoolID != st.SchoolID {
		return nil, fmt.Errorf("%w: that grade belongs to a different school", apperr.ErrInvalidInput)
	}
	if in.ClassSectionID != nil {
		section, err := s.academic.GetSection(ctx, *in.ClassSectionID)
		if err != nil {
			return nil, fmt.Errorf("%w: section not found", apperr.ErrInvalidInput)
		}
		if section.GradeLevelID != in.ToGradeLevelID || section.AcademicYearID != in.ToAcademicYearID {
			return nil, fmt.Errorf("%w: that section doesn't belong to the target grade/year", apperr.ErrInvalidInput)
		}
	}
	fs, err := s.fees.GetFeeStructureByGrade(ctx, st.SchoolID, in.ToAcademicYearID, in.ToGradeLevelID)
	if err != nil {
		return nil, fmt.Errorf("%w: no fee structure set up for %s in that academic year -- create one under Settings first", apperr.ErrInvalidInput, grade.Name)
	}

	// Carry-forward source: an explicit FromAcademicYearID, else whatever
	// account already exists for the target year (a same-year reassignment
	// has nothing new to carry forward -- the existing account is just
	// repointed at the new fee structure below).
	var source *domain.StudentFeeAccount
	if in.FromAcademicYearID != uuid.Nil {
		source, _ = s.fees.GetFeeAccountByStudent(ctx, in.StudentID, in.FromAcademicYearID)
	}

	var accountID uuid.UUID
	if existing, err := s.fees.GetFeeAccountByStudent(ctx, in.StudentID, in.ToAcademicYearID); err == nil {
		existing.FeeStructureID = fs.ID
		existing.TuitionFee = fs.TuitionFeeAnnual
		if err := s.fees.UpdateFeeAccount(ctx, existing); err != nil {
			return nil, err
		}
		accountID = existing.ID
	} else if apperr.IsNotFound(err) {
		fa := &domain.StudentFeeAccount{
			StudentID: in.StudentID, SchoolID: st.SchoolID, AcademicYearID: in.ToAcademicYearID,
			FeeStructureID: fs.ID, TuitionFee: fs.TuitionFeeAnnual,
		}
		if source != nil {
			if in.CarryForwardDiscount {
				fa.DiscountAmount = source.DiscountAmount
				fa.DiscountReason = source.DiscountReason
			}
			if in.CarryForwardVanFee {
				fa.VanFee = source.VanFee
			}
			fa.IsRTE = source.IsRTE
			if in.CarryForwardDues {
				balance, err := s.outstandingBalance(ctx, source)
				if err != nil {
					return nil, err
				}
				if balance > 0 {
					fa.PreviousYearDues = balance
				}
			}
		}
		if err := s.fees.CreateFeeAccount(ctx, fa); err != nil {
			return nil, err
		}
		accountID = fa.ID
	} else {
		return nil, err
	}

	if err := s.upsertEnrollment(ctx, st.SchoolID, in.StudentID, in.ToAcademicYearID, in.ClassSectionID); err != nil {
		return nil, err
	}
	// Close out the old year's enrollment (if any) so it reads as "completed"
	// rather than looking like an open, still-current enrollment.
	if in.FromAcademicYearID != uuid.Nil && in.FromAcademicYearID != in.ToAcademicYearID {
		if fromEnroll, err := s.enroll.GetByStudentYear(ctx, in.StudentID, in.FromAcademicYearID); err == nil {
			fromEnroll.Status = domain.EnrollmentStatusCompleted
			_ = s.enroll.Update(ctx, fromEnroll)
		}
	}

	return &MoveGradeResult{StudentID: in.StudentID, AccountID: accountID, GradeLevelID: in.ToGradeLevelID, GradeLevelName: grade.Name}, nil
}

func (s *Service) outstandingBalance(ctx context.Context, fa *domain.StudentFeeAccount) (int, error) {
	payments, err := s.fees.ListPayments(ctx, fa.ID)
	if err != nil {
		return 0, err
	}
	paid := 0
	for _, p := range payments {
		if !p.Voided {
			paid += p.Amount
		}
	}
	total := fa.TuitionFee - fa.DiscountAmount + fa.VanFee + fa.PreviousYearDues + fa.LateFee
	return total - paid, nil
}

func (s *Service) upsertEnrollment(ctx context.Context, schoolID, studentID, yearID uuid.UUID, sectionID *uuid.UUID) error {
	if existing, err := s.enroll.GetByStudentYear(ctx, studentID, yearID); err == nil {
		existing.ClassSectionID = sectionID
		existing.Status = domain.EnrollmentStatusActive
		return s.enroll.Update(ctx, existing)
	} else if !apperr.IsNotFound(err) {
		return err
	}
	e := &domain.Enrollment{
		StudentID: studentID, SchoolID: schoolID, AcademicYearID: yearID,
		ClassSectionID: sectionID, EnrollmentDate: time.Now(), Status: domain.EnrollmentStatusActive,
	}
	return s.enroll.Create(ctx, e)
}

// BulkMoveInput applies the same grade move to many students at once -- the
// normal shape of an actual promotion (a whole grade/section moving up
// together at year end). ClassSectionID is optional: leave nil to move
// students to the new grade without assigning a section yet (e.g. sections
// haven't been split for the new year), or set it to land the whole batch
// directly in one section -- the same "add a section, then move students
// into it" step MoveGrade already does for a single student.
type BulkMoveInput struct {
	StudentIDs           []uuid.UUID
	FromAcademicYearID   uuid.UUID
	ToAcademicYearID     uuid.UUID
	ToGradeLevelID       uuid.UUID
	ClassSectionID       *uuid.UUID
	CarryForwardDues     bool
	CarryForwardDiscount bool
	CarryForwardVanFee   bool
}

type BulkMoveRowResult struct {
	StudentID uuid.UUID `json:"student_id"`
	Success   bool      `json:"success"`
	Error     string    `json:"error,omitempty"`
}

// BulkMoveGrade runs MoveGrade for each student independently -- one
// student's failure (e.g. no fee structure yet for their current grade)
// doesn't block the rest of the batch, same pattern as bulk student import.
func (s *Service) BulkMoveGrade(ctx context.Context, in BulkMoveInput) []BulkMoveRowResult {
	results := make([]BulkMoveRowResult, 0, len(in.StudentIDs))
	for _, sid := range in.StudentIDs {
		_, err := s.MoveGrade(ctx, MoveGradeInput{
			StudentID: sid, FromAcademicYearID: in.FromAcademicYearID, ToAcademicYearID: in.ToAcademicYearID,
			ToGradeLevelID: in.ToGradeLevelID, ClassSectionID: in.ClassSectionID, CarryForwardDues: in.CarryForwardDues,
			CarryForwardDiscount: in.CarryForwardDiscount, CarryForwardVanFee: in.CarryForwardVanFee,
		})
		if err != nil {
			results = append(results, BulkMoveRowResult{StudentID: sid, Error: err.Error()})
			continue
		}
		results = append(results, BulkMoveRowResult{StudentID: sid, Success: true})
	}
	return results
}

// GetSection returns the student's currently assigned class section for
// academicYearID, or nil if they have no enrollment row yet, or it exists
// but isn't assigned to a section.
func (s *Service) GetSection(ctx context.Context, studentID, academicYearID uuid.UUID) (*uuid.UUID, error) {
	e, err := s.enroll.GetByStudentYear(ctx, studentID, academicYearID)
	if apperr.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return e.ClassSectionID, nil
}

// UpdateSectionInput reassigns a student's class section within one academic
// year, without touching their grade or fee account.
type UpdateSectionInput struct {
	StudentID      uuid.UUID
	AcademicYearID uuid.UUID
	ClassSectionID *uuid.UUID // nil clears the section assignment
}

func (s *Service) UpdateSection(ctx context.Context, in UpdateSectionInput) error {
	st, err := s.students.GetByID(ctx, in.StudentID)
	if err != nil {
		return err
	}
	if in.ClassSectionID != nil {
		section, err := s.academic.GetSection(ctx, *in.ClassSectionID)
		if err != nil {
			return fmt.Errorf("%w: section not found", apperr.ErrInvalidInput)
		}
		if section.AcademicYearID != in.AcademicYearID {
			return fmt.Errorf("%w: that section isn't in this academic year", apperr.ErrInvalidInput)
		}
		// If the student already has a fee account (i.e. a known grade) for
		// this year, the section must belong to that same grade -- prevents
		// a student ending up in "Grade 4B" while still billed as Grade 3.
		if fa, err := s.fees.GetFeeAccountByStudent(ctx, in.StudentID, in.AcademicYearID); err == nil {
			if fs, err := s.fees.GetFeeStructureByID(ctx, fa.FeeStructureID); err == nil && fs.GradeLevelID != section.GradeLevelID {
				return fmt.Errorf("%w: this section belongs to a different grade than the student is enrolled in", apperr.ErrInvalidInput)
			}
		}
	}
	return s.upsertEnrollment(ctx, st.SchoolID, in.StudentID, in.AcademicYearID, in.ClassSectionID)
}
