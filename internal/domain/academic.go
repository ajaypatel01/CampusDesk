package domain

import (
	"time"

	"github.com/google/uuid"
)

type AcademicYear struct {
	ID        uuid.UUID `json:"id"`
	SchoolID  uuid.UUID `json:"school_id"`
	Name      string    `json:"name"`
	StartDate time.Time `json:"start_date"`
	EndDate   time.Time `json:"end_date"`
	IsCurrent bool      `json:"is_current"`
	Timestamps
}

type GradeLevel struct {
	ID       uuid.UUID `json:"id"`
	SchoolID uuid.UUID `json:"school_id"`
	Name     string    `json:"name"`
	SortOrder int      `json:"sort_order"`
	Timestamps
}

type ClassSection struct {
	ID             uuid.UUID  `json:"id"`
	SchoolID       uuid.UUID  `json:"school_id"`
	AcademicYearID uuid.UUID  `json:"academic_year_id"`
	GradeLevelID   uuid.UUID  `json:"grade_level_id"`
	Name           string     `json:"name"`
	Capacity       int        `json:"capacity"`
	HomeroomTeacherID *uuid.UUID `json:"homeroom_teacher_id,omitempty"`
	Timestamps
}
