package domain

import (
	"time"

	"github.com/google/uuid"
)

type Subject struct {
	ID            uuid.UUID `json:"id"`
	SchoolID      uuid.UUID `json:"school_id"`
	GradeLevelID  uuid.UUID `json:"grade_level_id"`
	Name          string    `json:"name"`
	Code          string    `json:"code,omitempty"`
	MaxMarks      int       `json:"max_marks"`
	PassingMarks  int       `json:"passing_marks"`
	SortOrder     int       `json:"sort_order"`
	Timestamps
}

type Exam struct {
	ID             uuid.UUID  `json:"id"`
	SchoolID       uuid.UUID  `json:"school_id"`
	AcademicYearID uuid.UUID  `json:"academic_year_id"`
	GradeLevelID   uuid.UUID  `json:"grade_level_id"`
	Name           string     `json:"name"`
	ExamDate       *time.Time `json:"exam_date,omitempty"`
	WeightPercent  int        `json:"weight_percent"`
	IsPublished    bool       `json:"is_published"`
	Timestamps
}

type ExamMark struct {
	ID             uuid.UUID `json:"id"`
	ExamID         uuid.UUID `json:"exam_id"`
	StudentID      uuid.UUID `json:"student_id"`
	SubjectID      uuid.UUID `json:"subject_id"`
	MarksObtained  float64   `json:"marks_obtained"`
	MaxMarks       int       `json:"max_marks"`
	IsAbsent       bool      `json:"is_absent"`
	Remarks        string    `json:"remarks,omitempty"`
	Timestamps
}
