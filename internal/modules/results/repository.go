package results

import (
	"context"
	"errors"
	"fmt"

	"github.com/ajaypatel01/CampusDesk/internal/domain"
	"github.com/ajaypatel01/CampusDesk/internal/platform/database"
	apperr "github.com/ajaypatel01/CampusDesk/internal/platform/errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// ---- Subjects ----

func (r *Repository) CreateSubject(ctx context.Context, s *domain.Subject) error {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO subjects (school_id, grade_level_id, name, code, max_marks, passing_marks, sort_order)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
		RETURNING id, created_at, updated_at`,
		s.SchoolID, s.GradeLevelID, s.Name, s.Code, s.MaxMarks, s.PassingMarks, s.SortOrder,
	)
	if err := row.Scan(&s.ID, &s.CreatedAt, &s.UpdatedAt); err != nil {
		return database.MapError(err)
	}
	return nil
}

func (r *Repository) ListSubjects(ctx context.Context, schoolID, gradeLevelID uuid.UUID) ([]domain.Subject, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, school_id, grade_level_id, name, COALESCE(code,''), max_marks, passing_marks, sort_order, created_at, updated_at
		FROM subjects WHERE school_id=$1 AND grade_level_id=$2
		ORDER BY sort_order, name`, schoolID, gradeLevelID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []domain.Subject
	for rows.Next() {
		var s domain.Subject
		if err := rows.Scan(&s.ID, &s.SchoolID, &s.GradeLevelID, &s.Name, &s.Code,
			&s.MaxMarks, &s.PassingMarks, &s.SortOrder, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, s)
	}
	return items, rows.Err()
}

func (r *Repository) UpdateSubject(ctx context.Context, s *domain.Subject) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE subjects SET name=$2, code=$3, max_marks=$4, passing_marks=$5, sort_order=$6, updated_at=NOW()
		WHERE id=$1`, s.ID, s.Name, s.Code, s.MaxMarks, s.PassingMarks, s.SortOrder)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return apperr.ErrNotFound
	}
	return nil
}

func (r *Repository) DeleteSubject(ctx context.Context, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM subjects WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return apperr.ErrNotFound
	}
	return nil
}

// ---- Exams ----

func (r *Repository) CreateExam(ctx context.Context, e *domain.Exam) error {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO exams (school_id, academic_year_id, grade_level_id, name, exam_date, weight_percent, is_published)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
		RETURNING id, created_at, updated_at`,
		e.SchoolID, e.AcademicYearID, e.GradeLevelID, e.Name, e.ExamDate, e.WeightPercent, e.IsPublished,
	)
	if err := row.Scan(&e.ID, &e.CreatedAt, &e.UpdatedAt); err != nil {
		return database.MapError(err)
	}
	return nil
}

func (r *Repository) GetExamByID(ctx context.Context, id uuid.UUID) (*domain.Exam, error) {
	var e domain.Exam
	err := r.pool.QueryRow(ctx, `
		SELECT id, school_id, academic_year_id, grade_level_id, name, exam_date, weight_percent, is_published, created_at, updated_at
		FROM exams WHERE id=$1`, id,
	).Scan(&e.ID, &e.SchoolID, &e.AcademicYearID, &e.GradeLevelID, &e.Name, &e.ExamDate, &e.WeightPercent, &e.IsPublished, &e.CreatedAt, &e.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperr.ErrNotFound
	}
	return &e, err
}

func (r *Repository) ListExams(ctx context.Context, schoolID, yearID, gradeLevelID uuid.UUID) ([]domain.Exam, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, school_id, academic_year_id, grade_level_id, name, exam_date, weight_percent, is_published, created_at, updated_at
		FROM exams WHERE school_id=$1 AND academic_year_id=$2 AND grade_level_id=$3
		ORDER BY exam_date NULLS LAST, name`, schoolID, yearID, gradeLevelID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []domain.Exam
	for rows.Next() {
		var e domain.Exam
		if err := rows.Scan(&e.ID, &e.SchoolID, &e.AcademicYearID, &e.GradeLevelID, &e.Name, &e.ExamDate,
			&e.WeightPercent, &e.IsPublished, &e.CreatedAt, &e.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, e)
	}
	return items, rows.Err()
}

func (r *Repository) PublishExam(ctx context.Context, id uuid.UUID, publish bool) error {
	tag, err := r.pool.Exec(ctx, `UPDATE exams SET is_published=$2, updated_at=NOW() WHERE id=$1`, id, publish)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return apperr.ErrNotFound
	}
	return nil
}

// ---- Marks ----

func (r *Repository) UpsertMark(ctx context.Context, m *domain.ExamMark) error {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO exam_marks (exam_id, student_id, subject_id, marks_obtained, max_marks, is_absent, remarks)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
		ON CONFLICT (exam_id, student_id, subject_id)
		DO UPDATE SET marks_obtained=$4, max_marks=$5, is_absent=$6, remarks=$7, updated_at=NOW()
		RETURNING id, created_at, updated_at`,
		m.ExamID, m.StudentID, m.SubjectID, m.MarksObtained, m.MaxMarks, m.IsAbsent, m.Remarks,
	)
	if err := row.Scan(&m.ID, &m.CreatedAt, &m.UpdatedAt); err != nil {
		return database.MapError(err)
	}
	return nil
}

type MarksheetRow struct {
	SubjectName   string  `json:"subject_name"`
	SubjectCode   string  `json:"subject_code"`
	MaxMarks      int     `json:"max_marks"`
	PassingMarks  int     `json:"passing_marks"`
	MarksObtained float64 `json:"marks_obtained"`
	IsAbsent      bool    `json:"is_absent"`
	Percentage    float64 `json:"percentage"`
	Grade         string  `json:"grade"`
	GradePoint    float64 `json:"grade_point"`
	Status        string  `json:"status"` // Pass / Fail / Absent
}

type StudentMarksheet struct {
	StudentID      uuid.UUID      `json:"student_id"`
	StudentName    string         `json:"student_name"`
	StudentCode    string         `json:"student_code"`
	ExamID         uuid.UUID      `json:"exam_id"`
	ExamName       string         `json:"exam_name"`
	SchoolName     string         `json:"school_name"`
	GradeLevelName string         `json:"grade_level_name"`
	AcademicYear   string         `json:"academic_year"`
	Rows           []MarksheetRow `json:"rows"`
	TotalObtained  float64        `json:"total_obtained"`
	TotalMax       int            `json:"total_max"`
	Percentage     float64        `json:"percentage"`
	CGPA           float64        `json:"cgpa"`
	OverallGrade   string         `json:"overall_grade"`
	Result         string         `json:"result"` // Pass / Fail
}

func (r *Repository) GetStudentMarksheet(ctx context.Context, examID, studentID uuid.UUID) (*StudentMarksheet, error) {
	var ms StudentMarksheet
	ms.ExamID = examID
	ms.StudentID = studentID

	err := r.pool.QueryRow(ctx, `
		SELECT e.name, s.first_name||' '||s.last_name, s.student_code,
			sch.name, gl.name, ay.name
		FROM exams e
		JOIN schools sch ON sch.id = e.school_id
		JOIN grade_levels gl ON gl.id = e.grade_level_id
		JOIN academic_years ay ON ay.id = e.academic_year_id
		JOIN students s ON s.id = $2
		WHERE e.id = $1`, examID, studentID,
	).Scan(&ms.ExamName, &ms.StudentName, &ms.StudentCode, &ms.SchoolName, &ms.GradeLevelName, &ms.AcademicYear)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperr.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get marksheet header: %w", err)
	}

	rows, err := r.pool.Query(ctx, `
		SELECT sub.name, COALESCE(sub.code,''), em.max_marks, sub.passing_marks,
			em.marks_obtained, em.is_absent, COALESCE(em.remarks,'')
		FROM exam_marks em
		JOIN subjects sub ON sub.id = em.subject_id
		WHERE em.exam_id=$1 AND em.student_id=$2
		ORDER BY sub.sort_order, sub.name`, examID, studentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var totalObtained float64
	var totalMax int
	var totalGP float64

	for rows.Next() {
		var row MarksheetRow
		var remarks string
		if err := rows.Scan(&row.SubjectName, &row.SubjectCode, &row.MaxMarks, &row.PassingMarks,
			&row.MarksObtained, &row.IsAbsent, &remarks); err != nil {
			return nil, err
		}
		if row.IsAbsent {
			row.Status = "Absent"
			row.Grade = "AB"
		} else {
			row.Percentage = (row.MarksObtained / float64(row.MaxMarks)) * 100
			row.Grade, row.GradePoint = gradeFromPercent(row.Percentage)
			if row.MarksObtained >= float64(row.PassingMarks) {
				row.Status = "Pass"
			} else {
				row.Status = "Fail"
			}
			totalObtained += row.MarksObtained
			totalMax += row.MaxMarks
			totalGP += row.GradePoint
		}
		ms.Rows = append(ms.Rows, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	ms.TotalObtained = totalObtained
	ms.TotalMax = totalMax
	if totalMax > 0 {
		ms.Percentage = (totalObtained / float64(totalMax)) * 100
	}
	if len(ms.Rows) > 0 {
		ms.CGPA = totalGP / float64(len(ms.Rows))
	}
	ms.OverallGrade, _ = gradeFromPercent(ms.Percentage)

	ms.Result = "Pass"
	for _, row := range ms.Rows {
		if row.Status == "Fail" {
			ms.Result = "Fail"
			break
		}
	}
	return &ms, nil
}

func gradeFromPercent(pct float64) (grade string, gp float64) {
	switch {
	case pct >= 91:
		return "A1", 10.0
	case pct >= 81:
		return "A2", 9.0
	case pct >= 71:
		return "B1", 8.0
	case pct >= 61:
		return "B2", 7.0
	case pct >= 51:
		return "C1", 6.0
	case pct >= 41:
		return "C2", 5.0
	case pct >= 33:
		return "D", 4.0
	default:
		return "F", 0.0
	}
}
