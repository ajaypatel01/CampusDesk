package enrollment

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

func (r *Repository) Create(ctx context.Context, e *domain.Enrollment) error {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO enrollments (student_id, school_id, academic_year_id, class_section_id, enrollment_date, status)
		VALUES ($1,$2,$3,$4,$5,$6) RETURNING id, created_at, updated_at`,
		e.StudentID, e.SchoolID, e.AcademicYearID, e.ClassSectionID, e.EnrollmentDate, e.Status,
	)
	if err := row.Scan(&e.ID, &e.CreatedAt, &e.UpdatedAt); err != nil {
		return database.MapError(err)
	}
	return nil
}

func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Enrollment, error) {
	var e domain.Enrollment
	err := r.pool.QueryRow(ctx, `
		SELECT id, student_id, school_id, academic_year_id, class_section_id, enrollment_date, status, created_at, updated_at
		FROM enrollments WHERE id=$1`, id,
	).Scan(&e.ID, &e.StudentID, &e.SchoolID, &e.AcademicYearID, &e.ClassSectionID, &e.EnrollmentDate, &e.Status, &e.CreatedAt, &e.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperr.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get enrollment: %w", err)
	}
	return &e, nil
}

func (r *Repository) ListBySchoolYear(ctx context.Context, schoolID, yearID uuid.UUID, limit, offset int) ([]domain.Enrollment, int, error) {
	var total int
	if err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM enrollments WHERE school_id=$1 AND academic_year_id=$2`,
		schoolID, yearID,
	).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id, student_id, school_id, academic_year_id, class_section_id, enrollment_date, status, created_at, updated_at
		FROM enrollments WHERE school_id=$1 AND academic_year_id=$2
		ORDER BY enrollment_date DESC LIMIT $3 OFFSET $4`,
		schoolID, yearID, limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var items []domain.Enrollment
	for rows.Next() {
		var e domain.Enrollment
		if err := rows.Scan(&e.ID, &e.StudentID, &e.SchoolID, &e.AcademicYearID, &e.ClassSectionID, &e.EnrollmentDate, &e.Status, &e.CreatedAt, &e.UpdatedAt); err != nil {
			return nil, 0, err
		}
		items = append(items, e)
	}
	return items, total, rows.Err()
}

func (r *Repository) Update(ctx context.Context, e *domain.Enrollment) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE enrollments SET class_section_id=$2, status=$3, updated_at=NOW() WHERE id=$1`,
		e.ID, e.ClassSectionID, e.Status,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return apperr.ErrNotFound
	}
	return nil
}

// Attendance

func (r *Repository) UpsertAttendance(ctx context.Context, a *domain.AttendanceRecord) error {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO attendance_records (student_id, school_id, class_section_id, record_date, status, notes)
		VALUES ($1,$2,$3,$4,$5,$6)
		ON CONFLICT (student_id, record_date) DO UPDATE SET status=$5, notes=$6, class_section_id=$3, updated_at=NOW()
		RETURNING id, created_at, updated_at`,
		a.StudentID, a.SchoolID, a.ClassSectionID, a.RecordDate, a.Status, a.Notes,
	)
	if err := row.Scan(&a.ID, &a.CreatedAt, &a.UpdatedAt); err != nil {
		return database.MapError(err)
	}
	return nil
}

func (r *Repository) ListAttendance(ctx context.Context, schoolID uuid.UUID, date string, classSectionID *uuid.UUID) ([]domain.AttendanceRecord, error) {
	q := `
		SELECT id, student_id, school_id, class_section_id, record_date, status, notes, created_at, updated_at
		FROM attendance_records WHERE school_id=$1 AND record_date=$2`
	args := []interface{}{schoolID, date}
	if classSectionID != nil {
		q += ` AND class_section_id=$3`
		args = append(args, *classSectionID)
	}
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []domain.AttendanceRecord
	for rows.Next() {
		var a domain.AttendanceRecord
		if err := rows.Scan(&a.ID, &a.StudentID, &a.SchoolID, &a.ClassSectionID, &a.RecordDate, &a.Status, &a.Notes, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, a)
	}
	return items, rows.Err()
}
