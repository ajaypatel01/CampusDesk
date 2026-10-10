package guardian

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

func (r *Repository) Create(ctx context.Context, g *domain.Guardian) error {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO guardians (first_name, last_name, email, phone, relation, aadhar_number, user_id)
		VALUES ($1,$2,$3,$4,$5,$6,$7) RETURNING id, created_at, updated_at`,
		g.FirstName, g.LastName, g.Email, g.Phone, g.Relation, g.AadharNumber, g.UserID,
	)
	if err := row.Scan(&g.ID, &g.CreatedAt, &g.UpdatedAt); err != nil {
		return database.MapError(err)
	}
	return nil
}

// WardStudentIDs returns the students a parent-role user has portal access to,
// i.e. the students linked (via student_guardians) to the guardian record that
// carries this user's id.
func (r *Repository) WardStudentIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT sg.student_id
		FROM student_guardians sg
		JOIN guardians g ON g.id = sg.guardian_id
		JOIN students s ON s.id = sg.student_id
		WHERE g.user_id = $1 AND s.status <> 'duplicate'`, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (r *Repository) LinkStudent(ctx context.Context, studentID, guardianID uuid.UUID, isPrimary bool) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO student_guardians (student_id, guardian_id, is_primary)
		VALUES ($1,$2,$3) ON CONFLICT DO NOTHING`,
		studentID, guardianID, isPrimary,
	)
	return database.MapError(err)
}

func (r *Repository) ListByStudent(ctx context.Context, studentID uuid.UUID) ([]domain.Guardian, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT g.id, g.first_name, g.last_name, COALESCE(g.email,''), COALESCE(g.phone,''),
			COALESCE(g.relation,''), COALESCE(g.aadhar_number,''), sg.is_primary, g.created_at, g.updated_at
		FROM guardians g
		JOIN student_guardians sg ON sg.guardian_id = g.id
		WHERE sg.student_id = $1 ORDER BY sg.is_primary DESC, g.last_name`, studentID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []domain.Guardian
	for rows.Next() {
		var g domain.Guardian
		if err := rows.Scan(&g.ID, &g.FirstName, &g.LastName, &g.Email, &g.Phone, &g.Relation, &g.AadharNumber, &g.IsPrimary, &g.CreatedAt, &g.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, g)
	}
	return items, rows.Err()
}

func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Guardian, error) {
	var g domain.Guardian
	err := r.pool.QueryRow(ctx, `
		SELECT id, first_name, last_name, COALESCE(email,''), COALESCE(phone,''),
			COALESCE(relation,''), COALESCE(aadhar_number,''), created_at, updated_at
		FROM guardians WHERE id=$1`, id,
	).Scan(&g.ID, &g.FirstName, &g.LastName, &g.Email, &g.Phone, &g.Relation, &g.AadharNumber, &g.CreatedAt, &g.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperr.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get guardian: %w", err)
	}
	return &g, nil
}

// Update edits a guardian's contact details. It leaves user_id alone, so a
// parent's portal login keeps working after their details change.
func (r *Repository) Update(ctx context.Context, g *domain.Guardian) error {
	err := r.pool.QueryRow(ctx, `
		UPDATE guardians SET first_name=$2, last_name=$3, email=$4, phone=$5, relation=$6,
			aadhar_number=$7, updated_at=NOW()
		WHERE id=$1 RETURNING created_at, updated_at`,
		g.ID, g.FirstName, g.LastName, g.Email, g.Phone, g.Relation, g.AadharNumber,
	).Scan(&g.CreatedAt, &g.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return apperr.ErrNotFound
	}
	return database.MapError(err)
}

// StudentSchoolID returns the school a student belongs to.
func (r *Repository) StudentSchoolID(ctx context.Context, studentID uuid.UUID) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.pool.QueryRow(ctx, `SELECT school_id FROM students WHERE id=$1`, studentID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, apperr.ErrNotFound
	}
	return id, err
}

// GuardianSchools reports whether a guardian is linked to a student of
// schoolID, and whether they are linked to any student at all. Guardians
// have no school of their own -- they belong to a school through the
// students they are linked to.
func (r *Repository) GuardianSchools(ctx context.Context, guardianID, schoolID uuid.UUID) (inSchool, linked bool, err error) {
	err = r.pool.QueryRow(ctx, `
		SELECT
			EXISTS (SELECT 1 FROM student_guardians sg JOIN students s ON s.id = sg.student_id
			        WHERE sg.guardian_id = $1 AND s.school_id = $2),
			EXISTS (SELECT 1 FROM student_guardians WHERE guardian_id = $1)`,
		guardianID, schoolID,
	).Scan(&inSchool, &linked)
	return inSchool, linked, err
}
