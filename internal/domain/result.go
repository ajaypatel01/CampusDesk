package domain

import (
	"time"

	"github.com/google/uuid"
)

type Subject struct {
	ID           uuid.UUID `json:"id"`
	SchoolID     uuid.UUID `json:"school_id"`
	GradeLevelID uuid.UUID `json:"grade_level_id"`
	Name         string    `json:"name"`
	Code         string    `json:"code,omitempty"`
	MaxMarks     int       `json:"max_marks"`
	PassingMarks int       `json:"passing_marks"`
	SortOrder    int       `json:"sort_order"`
	// IsCoScholastic marks a subject (e.g. Computer, Music, Games) as graded but
	// excluded from the marksheet's overall total/percentage/CGPA and pass/fail
	// determination -- mirrors how co-scholastic subjects work on a typical
	// Indian school report card, as distinct from scored academic subjects.
	IsCoScholastic bool `json:"is_co_scholastic"`
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
	ID            uuid.UUID `json:"id"`
	ExamID        uuid.UUID `json:"exam_id"`
	StudentID     uuid.UUID `json:"student_id"`
	SubjectID     uuid.UUID `json:"subject_id"`
	MarksObtained float64   `json:"marks_obtained"`
	MaxMarks      int       `json:"max_marks"`
	IsAbsent      bool      `json:"is_absent"`
	Remarks       string    `json:"remarks,omitempty"`
	// Components carries per-component marks (e.g. "written": 55, "test": 18)
	// when the exam's grade uses one of the report-card templates. When set,
	// MarksObtained/MaxMarks are computed from it rather than taken as given.
	Components map[string]float64 `json:"components,omitempty"`
	Timestamps
}

// ExamMarkComponent is one graded component (Written, Note Book, Activity,
// Oral, Test, Project, Theory, ...) under an ExamMark. The parent ExamMark's
// MarksObtained/MaxMarks stay the computed sum, so code that only knows about
// a flat total (the existing single-exam marksheet/PDF) keeps working.
type ExamMarkComponent struct {
	ID           uuid.UUID `json:"id"`
	ExamMarkID   uuid.UUID `json:"exam_mark_id"`
	ComponentKey string    `json:"component_key"`
	Obtained     float64   `json:"obtained"`
	MaxMarks     int       `json:"max_marks"`
	Timestamps
}

// ReportCardDetails holds the handwritten-on-paper fields of a combined,
// multi-exam report card -- filled in once per student per academic year,
// not per exam.
type ReportCardDetails struct {
	ID             uuid.UUID `json:"id"`
	SchoolID       uuid.UUID `json:"school_id"`
	AcademicYearID uuid.UUID `json:"academic_year_id"`
	GradeLevelID   uuid.UUID `json:"grade_level_id"`
	StudentID      uuid.UUID `json:"student_id"`
	RollNo         string    `json:"roll_no,omitempty"`
	Attendance     string    `json:"attendance,omitempty"`
	Remark         string    `json:"remark,omitempty"`
	PromotedTo     string    `json:"promoted_to,omitempty"`
	MoralRemark    string    `json:"moral_remark,omitempty"`
	GKRemark       string    `json:"gk_remark,omitempty"`
	Timestamps
}

// DisciplineGrade is one Co-Scholastic/Discipline criterion grade (e.g. "Work
// Education" -> "A+") on the "middle" report-card template. Always typed by
// a teacher/admin, never computed from marks.
type DisciplineGrade struct {
	ID             uuid.UUID `json:"id"`
	SchoolID       uuid.UUID `json:"school_id"`
	AcademicYearID uuid.UUID `json:"academic_year_id"`
	StudentID      uuid.UUID `json:"student_id"`
	CriterionKey   string    `json:"criterion_key"`
	Grade          string    `json:"grade"`
	Timestamps
}
