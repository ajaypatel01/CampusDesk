package domain

import (
	"time"

	"github.com/google/uuid"
)

// StaffLeave is one recorded leave period for a staff member. Attendance is
// leave-based: a staff member is assumed present on every day unless a leave
// record covers that date.
type StaffLeave struct {
	ID             uuid.UUID `json:"id"`
	UserID         uuid.UUID `json:"user_id"`
	AcademicYearID uuid.UUID `json:"academic_year_id"`
	LeaveType      string    `json:"leave_type"` // "cl" or "unpaid"
	StartDate      time.Time `json:"start_date"`
	EndDate        time.Time `json:"end_date"`
	HalfDay        bool      `json:"half_day"` // only valid when StartDate == EndDate
	Reason         string    `json:"reason,omitempty"`
	Timestamps
}
