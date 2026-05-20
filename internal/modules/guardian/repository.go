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
		INSERT INTO guardians (first_name, last_name, email, phone, relation)
		VALUES ($1,$2,$3,$4,$5) RETURNING id, created_at, updated_at`,
		g.FirstName, g.LastName, g.Email, g.Phone, g.Relation,
	)
	if err := row.Scan(&g.ID, &g.CreatedAt, &g.UpdatedAt); err != nil {
		return database.MapError(err)
	}
	return nil
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
		SELECT g.id, g.first_name, g.last_name, g.email, g.phone, g.relation, sg.is_primary, g.created_at, g.updated_at
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
		if err := rows.Scan(&g.ID, &g.FirstName, &g.LastName, &g.Email, &g.Phone, &g.Relation, &g.IsPrimary, &g.CreatedAt, &g.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, g)
	}
	return items, rows.Err()
}

func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Guardian, error) {
	var g domain.Guardian
	err := r.pool.QueryRow(ctx, `
		SELECT id, first_name, last_name, email, phone, relation, created_at, updated_at
		FROM guardians WHERE id=$1`, id,
	).Scan(&g.ID, &g.FirstName, &g.LastName, &g.Email, &g.Phone, &g.Relation, &g.CreatedAt, &g.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperr.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get guardian: %w", err)
	}
	return &g, nil
}
