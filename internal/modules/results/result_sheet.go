package results

import (
	"context"
	"errors"
	"fmt"
	"sort"

	apperr "github.com/ajaypatel01/CampusDesk/internal/platform/errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v4"
)

// The result sheet is one exam's marks for a whole class: a row per student,
// a column per subject, plus each student's total, percentage, grade, result
// and rank. Totals come from scoreMarksheet, the same scoring as each
// student's own marksheet, so the two always agree.

type ResultSheetSubject struct {
	ID             uuid.UUID `json:"id"`
	Name           string    `json:"name"`
	MaxMarks       int       `json:"max_marks"`
	PassingMarks   int       `json:"passing_marks"`
	IsCoScholastic bool      `json:"is_co_scholastic"`
}

type ResultSheetCell struct {
	Obtained float64 `json:"obtained"`
	MaxMarks int     `json:"max_marks"`
	IsAbsent bool    `json:"is_absent"`
	Grade    string  `json:"grade"`
	Status   string  `json:"status"` // Pass / Fail / Absent / Graded
	// GradeLetter is the A/B/C/D grade of a letter-graded subject (no marks).
	GradeLetter string `json:"grade_letter,omitempty"`
}

type ResultSheetStudent struct {
	StudentID         uuid.UUID                  `json:"student_id"`
	StudentName       string                     `json:"student_name"`
	StudentCode       string                     `json:"student_code"`
	Marks             map[string]ResultSheetCell `json:"marks"` // by subject id
	HasMarks          bool                       `json:"has_marks"`
	TotalObtained     float64                    `json:"total_obtained"`
	TotalMax          int                        `json:"total_max"`
	Percentage        float64                    `json:"percentage"`
	Grade             string                     `json:"grade"`
	Result            string                     `json:"result"` // Pass / Fail, "" without marks
	IsTotalOverridden bool                       `json:"is_total_overridden"`
	// Rank by percentage among students with marks (1, 2, 2, 4 ...); 0 without marks.
	Rank int `json:"rank"`
}

type ResultSheet struct {
	ExamID         uuid.UUID            `json:"exam_id"`
	ExamName       string               `json:"exam_name"`
	GradeLevelName string               `json:"grade_level_name"`
	AcademicYear   string               `json:"academic_year"`
	IsPublished    bool                 `json:"is_published"`
	Subjects       []ResultSheetSubject `json:"subjects"`
	Students       []ResultSheetStudent `json:"students"`
}

// GetResultSheet builds the class result sheet for one exam. The class is the
// exam's grade: students whose fee account that year is for that grade (how
// the rest of the app places students), plus anyone with marks in the exam.
func (r *Repository) GetResultSheet(ctx context.Context, examID uuid.UUID) (*ResultSheet, error) {
	sheet := &ResultSheet{ExamID: examID, Subjects: []ResultSheetSubject{}, Students: []ResultSheetStudent{}}
	var schoolID, yearID, gradeID uuid.UUID
	err := r.pool.QueryRow(ctx, `
		SELECT e.name, e.is_published, e.school_id, e.academic_year_id, e.grade_level_id, gl.name, ay.name
		FROM exams e
		JOIN grade_levels gl ON gl.id = e.grade_level_id
		JOIN academic_years ay ON ay.id = e.academic_year_id
		WHERE e.id = $1`, examID,
	).Scan(&sheet.ExamName, &sheet.IsPublished, &schoolID, &yearID, &gradeID, &sheet.GradeLevelName, &sheet.AcademicYear)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperr.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get result sheet header: %w", err)
	}

	subjects, err := r.ListSubjects(ctx, schoolID, gradeID)
	if err != nil {
		return nil, err
	}
	for _, s := range subjects {
		sheet.Subjects = append(sheet.Subjects, ResultSheetSubject{
			ID: s.ID, Name: s.Name, MaxMarks: s.MaxMarks, PassingMarks: s.PassingMarks, IsCoScholastic: s.IsCoScholastic,
		})
	}

	studentRows, err := r.pool.Query(ctx, `
		SELECT s.id, TRIM(s.first_name || ' ' || s.last_name), s.student_code
		FROM students s
		WHERE s.id IN (
			SELECT sfa.student_id FROM student_fee_accounts sfa
			JOIN fee_structures fs ON fs.id = sfa.fee_structure_id
			WHERE sfa.school_id = $1 AND sfa.academic_year_id = $2 AND fs.grade_level_id = $3
			UNION
			SELECT em.student_id FROM exam_marks em WHERE em.exam_id = $4
		)
		ORDER BY s.first_name, s.last_name, s.student_code`, schoolID, yearID, gradeID, examID)
	if err != nil {
		return nil, err
	}
	index := map[uuid.UUID]int{}
	for studentRows.Next() {
		var st ResultSheetStudent
		if err := studentRows.Scan(&st.StudentID, &st.StudentName, &st.StudentCode); err != nil {
			studentRows.Close()
			return nil, err
		}
		st.Marks = map[string]ResultSheetCell{}
		index[st.StudentID] = len(sheet.Students)
		sheet.Students = append(sheet.Students, st)
	}
	studentRows.Close()
	if err := studentRows.Err(); err != nil {
		return nil, err
	}

	// Every mark in the exam, in one query, grouped into a marksheet per student.
	markRows, err := r.pool.Query(ctx, `
		SELECT em.student_id, sub.id, sub.name, COALESCE(sub.code,''), em.max_marks, sub.passing_marks,
			em.marks_obtained, em.is_absent, sub.is_co_scholastic, COALESCE(em.grade_letter,'')
		FROM exam_marks em
		JOIN subjects sub ON sub.id = em.subject_id
		WHERE em.exam_id = $1
		ORDER BY sub.sort_order, sub.name`, examID)
	if err != nil {
		return nil, err
	}
	sheets := map[uuid.UUID]*StudentMarksheet{}
	for markRows.Next() {
		var studentID uuid.UUID
		var row MarksheetRow
		if err := markRows.Scan(&studentID, &row.SubjectID, &row.SubjectName, &row.SubjectCode, &row.MaxMarks,
			&row.PassingMarks, &row.MarksObtained, &row.IsAbsent, &row.IsCoScholastic, &row.GradeLetter); err != nil {
			markRows.Close()
			return nil, err
		}
		ms := sheets[studentID]
		if ms == nil {
			ms = &StudentMarksheet{ExamID: examID, StudentID: studentID}
			sheets[studentID] = ms
		}
		ms.Rows = append(ms.Rows, row)
	}
	markRows.Close()
	if err := markRows.Err(); err != nil {
		return nil, err
	}

	overrides := map[uuid.UUID]float64{}
	orows, err := r.pool.Query(ctx, `SELECT student_id, total_obtained FROM marksheet_total_overrides WHERE exam_id = $1`, examID)
	if err != nil {
		return nil, err
	}
	for orows.Next() {
		var id uuid.UUID
		var total float64
		if err := orows.Scan(&id, &total); err != nil {
			orows.Close()
			return nil, err
		}
		overrides[id] = total
	}
	orows.Close()
	if err := orows.Err(); err != nil {
		return nil, err
	}

	for studentID, ms := range sheets {
		var override *float64
		if total, ok := overrides[studentID]; ok {
			override = &total
		}
		scoreMarksheet(ms, override)
		st := &sheet.Students[index[studentID]]
		st.HasMarks = true
		st.TotalObtained, st.TotalMax = ms.TotalObtained, ms.TotalMax
		st.Percentage, st.Grade, st.Result = ms.Percentage, ms.OverallGrade, ms.Result
		st.IsTotalOverridden = ms.IsTotalOverridden
		for _, row := range ms.Rows {
			st.Marks[row.SubjectID.String()] = ResultSheetCell{
				Obtained: row.MarksObtained, MaxMarks: row.MaxMarks, IsAbsent: row.IsAbsent, Grade: row.Grade, Status: row.Status,
				GradeLetter: row.GradeLetter,
			}
		}
	}

	rankStudents(sheet.Students)
	return sheet, nil
}

// rankStudents ranks students with marks by percentage, highest first, with
// ties sharing a rank (1, 2, 2, 4). Students without marks keep rank 0.
func rankStudents(students []ResultSheetStudent) {
	order := make([]int, 0, len(students))
	for i, st := range students {
		if st.HasMarks {
			order = append(order, i)
		}
	}
	sort.SliceStable(order, func(a, b int) bool {
		return students[order[a]].Percentage > students[order[b]].Percentage
	})
	for pos, i := range order {
		if pos > 0 && students[order[pos-1]].Percentage == students[i].Percentage {
			students[i].Rank = students[order[pos-1]].Rank
		} else {
			students[i].Rank = pos + 1
		}
	}
}
