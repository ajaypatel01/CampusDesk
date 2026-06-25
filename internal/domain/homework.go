package domain

import (
	"time"

	"github.com/google/uuid"
)

type HomeworkAssignment struct {
	ID             uuid.UUID  `json:"id"`
	SchoolID       uuid.UUID  `json:"school_id"`
	AcademicYearID uuid.UUID  `json:"academic_year_id"`
	GradeLevelID   uuid.UUID  `json:"grade_level_id"`
	ClassSectionID *uuid.UUID `json:"class_section_id,omitempty"`
	SubjectID      *uuid.UUID `json:"subject_id,omitempty"`
	Title          string     `json:"title"`
	Description    string     `json:"description,omitempty"`
	AssignedBy     *uuid.UUID `json:"assigned_by,omitempty"`
	AssignedDate   time.Time  `json:"assigned_date"`
	DueDate        time.Time  `json:"due_date"`
	Timestamps
}

type HomeworkSubmission struct {
	ID            uuid.UUID  `json:"id"`
	AssignmentID  uuid.UUID  `json:"assignment_id"`
	StudentID     uuid.UUID  `json:"student_id"`
	SubmittedDate *time.Time `json:"submitted_date,omitempty"`
	Status        string     `json:"status"`
	Remarks       string     `json:"remarks,omitempty"`
	Timestamps
}
