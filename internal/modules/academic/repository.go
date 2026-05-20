package academic

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

// Academic years

func (r *Repository) CreateYear(ctx context.Context, y *domain.AcademicYear) error {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO academic_years (school_id, name, start_date, end_date, is_current)
		VALUES ($1,$2,$3,$4,$5) RETURNING id, created_at, updated_at`,
		y.SchoolID, y.Name, y.StartDate, y.EndDate, y.IsCurrent,
	)
	if err := row.Scan(&y.ID, &y.CreatedAt, &y.UpdatedAt); err != nil {
		return database.MapError(err)
	}
	return nil
}

func (r *Repository) ListYears(ctx context.Context, schoolID uuid.UUID) ([]domain.AcademicYear, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, school_id, name, start_date, end_date, is_current, created_at, updated_at
		FROM academic_years WHERE school_id=$1 ORDER BY start_date DESC`, schoolID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []domain.AcademicYear
	for rows.Next() {
		var y domain.AcademicYear
		if err := rows.Scan(&y.ID, &y.SchoolID, &y.Name, &y.StartDate, &y.EndDate, &y.IsCurrent, &y.CreatedAt, &y.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, y)
	}
	return items, rows.Err()
}

// Grade levels

func (r *Repository) CreateGrade(ctx context.Context, g *domain.GradeLevel) error {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO grade_levels (school_id, name, sort_order) VALUES ($1,$2,$3)
		RETURNING id, created_at, updated_at`, g.SchoolID, g.Name, g.SortOrder,
	)
	if err := row.Scan(&g.ID, &g.CreatedAt, &g.UpdatedAt); err != nil {
		return database.MapError(err)
	}
	return nil
}

func (r *Repository) ListGrades(ctx context.Context, schoolID uuid.UUID) ([]domain.GradeLevel, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, school_id, name, sort_order, created_at, updated_at
		FROM grade_levels WHERE school_id=$1 ORDER BY sort_order, name`, schoolID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []domain.GradeLevel
	for rows.Next() {
		var g domain.GradeLevel
		if err := rows.Scan(&g.ID, &g.SchoolID, &g.Name, &g.SortOrder, &g.CreatedAt, &g.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, g)
	}
	return items, rows.Err()
}

// Class sections

func (r *Repository) CreateSection(ctx context.Context, c *domain.ClassSection) error {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO class_sections (school_id, academic_year_id, grade_level_id, name, capacity, homeroom_teacher_id)
		VALUES ($1,$2,$3,$4,$5,$6) RETURNING id, created_at, updated_at`,
		c.SchoolID, c.AcademicYearID, c.GradeLevelID, c.Name, c.Capacity, c.HomeroomTeacherID,
	)
	if err := row.Scan(&c.ID, &c.CreatedAt, &c.UpdatedAt); err != nil {
		return database.MapError(err)
	}
	return nil
}

func (r *Repository) ListSections(ctx context.Context, schoolID, yearID uuid.UUID) ([]domain.ClassSection, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, school_id, academic_year_id, grade_level_id, name, capacity, homeroom_teacher_id, created_at, updated_at
		FROM class_sections WHERE school_id=$1 AND academic_year_id=$2 ORDER BY name`,
		schoolID, yearID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []domain.ClassSection
	for rows.Next() {
		var c domain.ClassSection
		if err := rows.Scan(&c.ID, &c.SchoolID, &c.AcademicYearID, &c.GradeLevelID, &c.Name, &c.Capacity, &c.HomeroomTeacherID, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, c)
	}
	return items, rows.Err()
}

func (r *Repository) GetSection(ctx context.Context, id uuid.UUID) (*domain.ClassSection, error) {
	var c domain.ClassSection
	err := r.pool.QueryRow(ctx, `
		SELECT id, school_id, academic_year_id, grade_level_id, name, capacity, homeroom_teacher_id, created_at, updated_at
		FROM class_sections WHERE id=$1`, id,
	).Scan(&c.ID, &c.SchoolID, &c.AcademicYearID, &c.GradeLevelID, &c.Name, &c.Capacity, &c.HomeroomTeacherID, &c.CreatedAt, &c.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperr.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get section: %w", err)
	}
	return &c, nil
}
