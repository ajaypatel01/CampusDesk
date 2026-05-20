package school

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

func (r *Repository) Create(ctx context.Context, s *domain.School) error {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO schools (name, code, address, phone, email)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at`,
		s.Name, s.Code, s.Address, s.Phone, s.Email,
	)
	if err := row.Scan(&s.ID, &s.CreatedAt, &s.UpdatedAt); err != nil {
		return database.MapError(err)
	}
	return nil
}

func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*domain.School, error) {
	s := &domain.School{}
	err := r.pool.QueryRow(ctx, `
		SELECT id, name, code, address, phone, email, created_at, updated_at
		FROM schools WHERE id = $1`, id,
	).Scan(&s.ID, &s.Name, &s.Code, &s.Address, &s.Phone, &s.Email, &s.CreatedAt, &s.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperr.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get school: %w", err)
	}
	return s, nil
}

func (r *Repository) List(ctx context.Context, limit, offset int) ([]domain.School, int, error) {
	var total int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM schools`).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.pool.Query(ctx, `
		SELECT id, name, code, address, phone, email, created_at, updated_at
		FROM schools ORDER BY name LIMIT $1 OFFSET $2`, limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var schools []domain.School
	for rows.Next() {
		var s domain.School
		if err := rows.Scan(&s.ID, &s.Name, &s.Code, &s.Address, &s.Phone, &s.Email, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, 0, err
		}
		schools = append(schools, s)
	}
	return schools, total, rows.Err()
}

func (r *Repository) Update(ctx context.Context, s *domain.School) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE schools SET name=$2, code=$3, address=$4, phone=$5, email=$6, updated_at=NOW()
		WHERE id=$1`,
		s.ID, s.Name, s.Code, s.Address, s.Phone, s.Email,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return apperr.ErrNotFound
	}
	return nil
}

func (r *Repository) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM schools WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return apperr.ErrNotFound
	}
	return nil
}
