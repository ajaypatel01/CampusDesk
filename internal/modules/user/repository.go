package user

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

func (r *Repository) Create(ctx context.Context, u *domain.User) error {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO users (school_id, email, password_hash, first_name, last_name, role, is_active)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
		RETURNING id, created_at, updated_at`,
		u.SchoolID, u.Email, u.PasswordHash, u.FirstName, u.LastName, u.Role, u.IsActive,
	)
	if err := row.Scan(&u.ID, &u.CreatedAt, &u.UpdatedAt); err != nil {
		return database.MapError(err)
	}
	return nil
}

func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	return r.scanOne(ctx, `SELECT id, school_id, email, password_hash, first_name, last_name, role, is_active, created_at, updated_at FROM users WHERE id=$1`, id)
}

func (r *Repository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	return r.scanOne(ctx, `SELECT id, school_id, email, password_hash, first_name, last_name, role, is_active, created_at, updated_at FROM users WHERE email=$1`, email)
}

func (r *Repository) List(ctx context.Context, schoolID *uuid.UUID, limit, offset int) ([]domain.User, int, error) {
	var total int
	var rows pgx.Rows
	var err error

	if schoolID != nil {
		if err = r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM users WHERE school_id=$1`, *schoolID).Scan(&total); err != nil {
			return nil, 0, err
		}
		rows, err = r.pool.Query(ctx, `
			SELECT id, school_id, email, password_hash, first_name, last_name, role, is_active, created_at, updated_at
			FROM users WHERE school_id=$1 ORDER BY last_name, first_name LIMIT $2 OFFSET $3`,
			*schoolID, limit, offset,
		)
	} else {
		if err = r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&total); err != nil {
			return nil, 0, err
		}
		rows, err = r.pool.Query(ctx, `
			SELECT id, school_id, email, password_hash, first_name, last_name, role, is_active, created_at, updated_at
			FROM users ORDER BY last_name, first_name LIMIT $1 OFFSET $2`, limit, offset,
		)
	}
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	return r.collect(rows, total)
}

func (r *Repository) collect(rows pgx.Rows, total int) ([]domain.User, int, error) {
	var users []domain.User
	for rows.Next() {
		u, err := scanRow(rows)
		if err != nil {
			return nil, 0, err
		}
		users = append(users, *u)
	}
	return users, total, rows.Err()
}

func (r *Repository) scanOne(ctx context.Context, q string, arg interface{}) (*domain.User, error) {
	row := r.pool.QueryRow(ctx, q, arg)
	u, err := scanRow(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperr.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan user: %w", err)
	}
	return u, nil
}

type scannable interface {
	Scan(dest ...interface{}) error
}

func scanRow(row scannable) (*domain.User, error) {
	var u domain.User
	err := row.Scan(&u.ID, &u.SchoolID, &u.Email, &u.PasswordHash, &u.FirstName, &u.LastName, &u.Role, &u.IsActive, &u.CreatedAt, &u.UpdatedAt)
	return &u, err
}
