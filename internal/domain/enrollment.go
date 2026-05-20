package domain

import (
	"time"

	"github.com/google/uuid"
)

type Enrollment struct {
	ID             uuid.UUID        `json:"id"`
	StudentID      uuid.UUID        `json:"student_id"`
	SchoolID       uuid.UUID        `json:"school_id"`
	AcademicYearID uuid.UUID        `json:"academic_year_id"`
	ClassSectionID *uuid.UUID       `json:"class_section_id,omitempty"`
	EnrollmentDate time.Time        `json:"enrollment_date"`
	Status         EnrollmentStatus `json:"status"`
	Timestamps
}

type AttendanceRecord struct {
	ID             uuid.UUID        `json:"id"`
	StudentID      uuid.UUID        `json:"student_id"`
	SchoolID       uuid.UUID        `json:"school_id"`
	ClassSectionID *uuid.UUID       `json:"class_section_id,omitempty"`
	RecordDate     time.Time        `json:"record_date"`
	Status         AttendanceStatus `json:"status"`
	Notes          string           `json:"notes,omitempty"`
	Timestamps
}
