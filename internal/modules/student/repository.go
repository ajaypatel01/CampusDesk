package student

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

type ListFilter struct {
	SchoolID uuid.UUID
	Status   string
	Search   string
}

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Create(ctx context.Context, s *domain.Student) error {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO students (school_id, student_code, first_name, last_name, date_of_birth, gender, email, phone, address, status)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		RETURNING id, created_at, updated_at`,
		s.SchoolID, s.StudentCode, s.FirstName, s.LastName, s.DateOfBirth, s.Gender, s.Email, s.Phone, s.Address, s.Status,
	)
	if err := row.Scan(&s.ID, &s.CreatedAt, &s.UpdatedAt); err != nil {
		return database.MapError(err)
	}
	return nil
}

func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Student, error) {
	return r.scanOne(ctx, `SELECT id, school_id, student_code, first_name, last_name, date_of_birth, gender, email, phone, address, status, created_at, updated_at FROM students WHERE id=$1`, id)
}

func (r *Repository) List(ctx context.Context, f ListFilter, limit, offset int) ([]domain.Student, int, error) {
	args := []interface{}{f.SchoolID}
	where := "WHERE school_id = $1"
	argN := 2

	if f.Status != "" {
		where += fmt.Sprintf(" AND status = $%d", argN)
		args = append(args, f.Status)
		argN++
	}
	if f.Search != "" {
		where += fmt.Sprintf(" AND (first_name ILIKE $%d OR last_name ILIKE $%d OR student_code ILIKE $%d)", argN, argN, argN)
		args = append(args, "%"+f.Search+"%")
		argN++
	}

	var total int
	countQ := "SELECT COUNT(*) FROM students " + where
	if err := r.pool.QueryRow(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	q := `SELECT id, school_id, student_code, first_name, last_name, date_of_birth, gender, email, phone, address, status, created_at, updated_at FROM students ` +
		where + fmt.Sprintf(" ORDER BY last_name, first_name LIMIT $%d OFFSET $%d", argN, argN+1)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var items []domain.Student
	for rows.Next() {
		s, err := scanRow(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, *s)
	}
	return items, total, rows.Err()
}

func (r *Repository) Update(ctx context.Context, s *domain.Student) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE students SET student_code=$2, first_name=$3, last_name=$4, date_of_birth=$5, gender=$6,
			email=$7, phone=$8, address=$9, status=$10, updated_at=NOW()
		WHERE id=$1`,
		s.ID, s.StudentCode, s.FirstName, s.LastName, s.DateOfBirth, s.Gender, s.Email, s.Phone, s.Address, s.Status,
	)
	if err != nil {
		return database.MapError(err)
	}
	if tag.RowsAffected() == 0 {
		return apperr.ErrNotFound
	}
	return nil
}

func (r *Repository) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM students WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return apperr.ErrNotFound
	}
	return nil
}

func (r *Repository) scanOne(ctx context.Context, q string, id uuid.UUID) (*domain.Student, error) {
	row := r.pool.QueryRow(ctx, q, id)
	s, err := scanRow(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperr.ErrNotFound
	}
	return s, err
}

type scannable interface {
	Scan(dest ...interface{}) error
}

func scanRow(row scannable) (*domain.Student, error) {
	var s domain.Student
	err := row.Scan(&s.ID, &s.SchoolID, &s.StudentCode, &s.FirstName, &s.LastName, &s.DateOfBirth, &s.Gender, &s.Email, &s.Phone, &s.Address, &s.Status, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &s, nil
}
