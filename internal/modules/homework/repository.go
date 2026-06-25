package homework

import (
	"context"
	"errors"

	"github.com/ajaypatel01/CampusDesk/internal/domain"
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

// ---- Assignments ----

func (r *Repository) CreateAssignment(ctx context.Context, a *domain.HomeworkAssignment) error {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO homework_assignments
			(school_id, academic_year_id, grade_level_id, class_section_id, subject_id,
			 title, description, assigned_by, assigned_date, due_date)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		RETURNING id, created_at, updated_at`,
		a.SchoolID, a.AcademicYearID, a.GradeLevelID, a.ClassSectionID, a.SubjectID,
		a.Title, a.Description, a.AssignedBy, a.AssignedDate, a.DueDate,
	)
	return row.Scan(&a.ID, &a.CreatedAt, &a.UpdatedAt)
}

func (r *Repository) GetAssignment(ctx context.Context, id uuid.UUID) (*domain.HomeworkAssignment, error) {
	var a domain.HomeworkAssignment
	err := r.pool.QueryRow(ctx, `
		SELECT id, school_id, academic_year_id, grade_level_id, class_section_id, subject_id,
			title, COALESCE(description,''), assigned_by, assigned_date, due_date, created_at, updated_at
		FROM homework_assignments WHERE id=$1`, id,
	).Scan(&a.ID, &a.SchoolID, &a.AcademicYearID, &a.GradeLevelID, &a.ClassSectionID, &a.SubjectID,
		&a.Title, &a.Description, &a.AssignedBy, &a.AssignedDate, &a.DueDate, &a.CreatedAt, &a.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperr.ErrNotFound
	}
	return &a, err
}

func (r *Repository) ListAssignments(ctx context.Context, schoolID, yearID, gradeLevelID uuid.UUID) ([]domain.HomeworkAssignment, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, school_id, academic_year_id, grade_level_id, class_section_id, subject_id,
			title, COALESCE(description,''), assigned_by, assigned_date, due_date, created_at, updated_at
		FROM homework_assignments
		WHERE school_id=$1 AND academic_year_id=$2 AND grade_level_id=$3
		ORDER BY due_date DESC`, schoolID, yearID, gradeLevelID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []domain.HomeworkAssignment
	for rows.Next() {
		var a domain.HomeworkAssignment
		if err := rows.Scan(&a.ID, &a.SchoolID, &a.AcademicYearID, &a.GradeLevelID, &a.ClassSectionID, &a.SubjectID,
			&a.Title, &a.Description, &a.AssignedBy, &a.AssignedDate, &a.DueDate, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, a)
	}
	return items, rows.Err()
}

func (r *Repository) DeleteAssignment(ctx context.Context, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM homework_assignments WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return apperr.ErrNotFound
	}
	return nil
}

// ---- Submissions ----

func (r *Repository) UpsertSubmission(ctx context.Context, s *domain.HomeworkSubmission) error {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO homework_submissions (assignment_id, student_id, submitted_date, status, remarks)
		VALUES ($1,$2,$3,$4,$5)
		ON CONFLICT (assignment_id, student_id)
		DO UPDATE SET submitted_date=$3, status=$4, remarks=$5, updated_at=NOW()
		RETURNING id, created_at, updated_at`,
		s.AssignmentID, s.StudentID, s.SubmittedDate, s.Status, s.Remarks,
	)
	return row.Scan(&s.ID, &s.CreatedAt, &s.UpdatedAt)
}

func (r *Repository) ListSubmissions(ctx context.Context, assignmentID uuid.UUID) ([]domain.HomeworkSubmission, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, assignment_id, student_id, submitted_date, status, COALESCE(remarks,''), created_at, updated_at
		FROM homework_submissions WHERE assignment_id=$1
		ORDER BY created_at`, assignmentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []domain.HomeworkSubmission
	for rows.Next() {
		var s domain.HomeworkSubmission
		if err := rows.Scan(&s.ID, &s.AssignmentID, &s.StudentID, &s.SubmittedDate,
			&s.Status, &s.Remarks, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, s)
	}
	return items, rows.Err()
}

func (r *Repository) GetStudentSubmissions(ctx context.Context, studentID, yearID uuid.UUID) ([]domain.HomeworkSubmission, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT hs.id, hs.assignment_id, hs.student_id, hs.submitted_date, hs.status, COALESCE(hs.remarks,''), hs.created_at, hs.updated_at
		FROM homework_submissions hs
		JOIN homework_assignments ha ON ha.id = hs.assignment_id
		WHERE hs.student_id=$1 AND ha.academic_year_id=$2
		ORDER BY ha.due_date DESC`, studentID, yearID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []domain.HomeworkSubmission
	for rows.Next() {
		var s domain.HomeworkSubmission
		if err := rows.Scan(&s.ID, &s.AssignmentID, &s.StudentID, &s.SubmittedDate,
			&s.Status, &s.Remarks, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, s)
	}
	return items, rows.Err()
}
