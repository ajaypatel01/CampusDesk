package domain

import (
	"time"

	"github.com/google/uuid"
)

type Result struct {
	ID             uuid.UUID       `json:"id"`
	StudentID      uuid.UUID       `json:"student_id"`
	SchoolID       uuid.UUID       `json:"school_id"`
	AcademicYearID uuid.UUID       `json:"academic_year_id"`
	ClassSectionID *uuid.UUID      `json:"class_section_id,omitempty"`
	ExamName       string          `json:"exam_name"`
	TotalMarks     float64         `json:"total_marks"`
	MaxTotalMarks  float64         `json:"max_total_marks"`
	Percentage     float64         `json:"percentage"`
	FinalGrade     string          `json:"final_grade,omitempty"`
	Remarks        string          `json:"remarks,omitempty"`
	ResultDate     time.Time       `json:"result_date"`
	Status         ResultStatus    `json:"status"`
	Subjects       []ResultSubject `json:"subjects"`
	Timestamps
}

type ResultSubject struct {
	ID            uuid.UUID `json:"id"`
	ResultID      uuid.UUID `json:"result_id"`
	SubjectName   string    `json:"subject_name"`
	MarksObtained float64   `json:"marks_obtained"`
	MaxMarks      float64   `json:"max_marks"`
	Grade         string    `json:"grade,omitempty"`
	Remarks       string    `json:"remarks,omitempty"`
	SortOrder     int       `json:"sort_order"`
	Timestamps
}
