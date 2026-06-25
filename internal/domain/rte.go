package domain

import "github.com/google/uuid"

type RTEQuota struct {
	ID                          uuid.UUID `json:"id"`
	SchoolID                    uuid.UUID `json:"school_id"`
	AcademicYearID              uuid.UUID `json:"academic_year_id"`
	GradeLevelID                uuid.UUID `json:"grade_level_id"`
	TotalSeats                  int       `json:"total_seats"`
	GovtReimbursementPerStudent int       `json:"govt_reimbursement_per_student"`
	Notes                       string    `json:"notes,omitempty"`
	Timestamps
}
