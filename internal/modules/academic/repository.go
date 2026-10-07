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
		SELECT id, school_id, name, sort_order, report_card_template, created_at, updated_at
		FROM grade_levels WHERE school_id=$1 ORDER BY sort_order, name`, schoolID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []domain.GradeLevel
	for rows.Next() {
		var g domain.GradeLevel
		if err := rows.Scan(&g.ID, &g.SchoolID, &g.Name, &g.SortOrder, &g.ReportCardTemplate, &g.CreatedAt, &g.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, g)
	}
	return items, rows.Err()
}

// UpdateGrade updates a grade level's name, sort order, and report-card
// template. Name/sort order were previously create-only.
func (r *Repository) UpdateGrade(ctx context.Context, g *domain.GradeLevel) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE grade_levels SET name=$2, sort_order=$3, report_card_template=$4, updated_at=NOW()
		WHERE id=$1`, g.ID, g.Name, g.SortOrder, g.ReportCardTemplate)
	if err != nil {
		return database.MapError(err)
	}
	if tag.RowsAffected() == 0 {
		return apperr.ErrNotFound
	}
	return nil
}

// GetGrade returns a single grade level by id.
func (r *Repository) GetGrade(ctx context.Context, id uuid.UUID) (*domain.GradeLevel, error) {
	var g domain.GradeLevel
	err := r.pool.QueryRow(ctx, `
		SELECT id, school_id, name, sort_order, report_card_template, created_at, updated_at
		FROM grade_levels WHERE id=$1`, id,
	).Scan(&g.ID, &g.SchoolID, &g.Name, &g.SortOrder, &g.ReportCardTemplate, &g.CreatedAt, &g.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperr.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &g, nil
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
		SELECT id, school_id, academic_year_id, grade_level_id, name, capacity, homeroom_teacher_id, created_at, updated_at,
			COALESCE(ARRAY(SELECT v.user_id::text FROM class_section_vice_teachers v WHERE v.class_section_id = class_sections.id ORDER BY v.created_at), '{}')
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
		var vice []string
		if err := rows.Scan(&c.ID, &c.SchoolID, &c.AcademicYearID, &c.GradeLevelID, &c.Name, &c.Capacity, &c.HomeroomTeacherID, &c.CreatedAt, &c.UpdatedAt, &vice); err != nil {
			return nil, err
		}
		c.ViceTeacherIDs = parseUUIDs(vice)
		items = append(items, c)
	}
	return items, rows.Err()
}

func (r *Repository) UpdateSection(ctx context.Context, c *domain.ClassSection) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE class_sections SET name=$2, capacity=$3, homeroom_teacher_id=$4, updated_at=NOW()
		WHERE id=$1`, c.ID, c.Name, c.Capacity, c.HomeroomTeacherID)
	if err != nil {
		return database.MapError(err)
	}
	if tag.RowsAffected() == 0 {
		return apperr.ErrNotFound
	}
	return nil
}

func (r *Repository) GetSection(ctx context.Context, id uuid.UUID) (*domain.ClassSection, error) {
	var c domain.ClassSection
	var vice []string
	err := r.pool.QueryRow(ctx, `
		SELECT id, school_id, academic_year_id, grade_level_id, name, capacity, homeroom_teacher_id, created_at, updated_at,
			COALESCE(ARRAY(SELECT v.user_id::text FROM class_section_vice_teachers v WHERE v.class_section_id = class_sections.id ORDER BY v.created_at), '{}')
		FROM class_sections WHERE id=$1`, id,
	).Scan(&c.ID, &c.SchoolID, &c.AcademicYearID, &c.GradeLevelID, &c.Name, &c.Capacity, &c.HomeroomTeacherID, &c.CreatedAt, &c.UpdatedAt, &vice)
	c.ViceTeacherIDs = parseUUIDs(vice)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperr.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get section: %w", err)
	}
	return &c, nil
}

func parseUUIDs(ss []string) []uuid.UUID {
	out := make([]uuid.UUID, 0, len(ss))
	for _, s := range ss {
		if id, err := uuid.Parse(s); err == nil {
			out = append(out, id)
		}
	}
	return out
}

// SetSectionViceTeachers replaces a section's vice class teachers.
func (r *Repository) SetSectionViceTeachers(ctx context.Context, sectionID uuid.UUID, userIDs []uuid.UUID) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `DELETE FROM class_section_vice_teachers WHERE class_section_id=$1`, sectionID); err != nil {
		return err
	}
	for _, id := range userIDs {
		if _, err := tx.Exec(ctx, `INSERT INTO class_section_vice_teachers (class_section_id, user_id) VALUES ($1,$2)`, sectionID, id); err != nil {
			return database.MapError(err)
		}
	}
	return tx.Commit(ctx)
}

// TeachersInSchool returns which of userIDs are active teachers at schoolID.
func (r *Repository) TeachersInSchool(ctx context.Context, schoolID uuid.UUID, userIDs []uuid.UUID) (map[uuid.UUID]bool, error) {
	ids := make([]string, len(userIDs))
	for i, id := range userIDs {
		ids[i] = id.String()
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id FROM users WHERE id = ANY($1::uuid[]) AND school_id = $2 AND role = 'teacher' AND is_active`, ids, schoolID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ok := map[uuid.UUID]bool{}
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ok[id] = true
	}
	return ok, rows.Err()
}
