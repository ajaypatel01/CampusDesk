package payroll

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

const leaveSelect = `
	SELECT id, user_id, academic_year_id, leave_type, start_date, end_date, COALESCE(reason,''), created_at, updated_at
	FROM staff_leaves`

func scanLeave(row interface{ Scan(...interface{}) error }) (*domain.StaffLeave, error) {
	var l domain.StaffLeave
	if err := row.Scan(&l.ID, &l.UserID, &l.AcademicYearID, &l.LeaveType, &l.StartDate, &l.EndDate, &l.Reason, &l.CreatedAt, &l.UpdatedAt); err != nil {
		return nil, err
	}
	return &l, nil
}

func (r *Repository) CreateLeave(ctx context.Context, l *domain.StaffLeave) error {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO staff_leaves (user_id, academic_year_id, leave_type, start_date, end_date, reason)
		VALUES ($1,$2,$3,$4,$5,$6)
		RETURNING id, created_at, updated_at`,
		l.UserID, l.AcademicYearID, l.LeaveType, l.StartDate, l.EndDate, l.Reason,
	)
	if err := row.Scan(&l.ID, &l.CreatedAt, &l.UpdatedAt); err != nil {
		return database.MapError(err)
	}
	return nil
}

func (r *Repository) UpdateLeave(ctx context.Context, l *domain.StaffLeave) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE staff_leaves SET leave_type=$2, start_date=$3, end_date=$4, reason=$5, updated_at=NOW()
		WHERE id=$1`,
		l.ID, l.LeaveType, l.StartDate, l.EndDate, l.Reason,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return apperr.ErrNotFound
	}
	return nil
}

func (r *Repository) DeleteLeave(ctx context.Context, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM staff_leaves WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return apperr.ErrNotFound
	}
	return nil
}

func (r *Repository) GetLeave(ctx context.Context, id uuid.UUID) (*domain.StaffLeave, error) {
	l, err := scanLeave(r.pool.QueryRow(ctx, leaveSelect+` WHERE id=$1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperr.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get leave: %w", err)
	}
	return l, nil
}

func (r *Repository) ListLeavesByUserYear(ctx context.Context, userID, academicYearID uuid.UUID) ([]domain.StaffLeave, error) {
	rows, err := r.pool.Query(ctx, leaveSelect+` WHERE user_id=$1 AND academic_year_id=$2 ORDER BY start_date`, userID, academicYearID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var leaves []domain.StaffLeave
	for rows.Next() {
		l, err := scanLeave(rows)
		if err != nil {
			return nil, err
		}
		leaves = append(leaves, *l)
	}
	return leaves, rows.Err()
}

// ListLeavesBySchoolYear returns every leave for staff belonging to schoolID within
// academicYearID, used to compute payroll for the whole school in one query.
func (r *Repository) ListLeavesBySchoolYear(ctx context.Context, schoolID, academicYearID uuid.UUID) ([]domain.StaffLeave, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT sl.id, sl.user_id, sl.academic_year_id, sl.leave_type, sl.start_date, sl.end_date, COALESCE(sl.reason,''), sl.created_at, sl.updated_at
		FROM staff_leaves sl
		JOIN users u ON u.id = sl.user_id
		WHERE u.school_id=$1 AND sl.academic_year_id=$2
		ORDER BY sl.start_date`,
		schoolID, academicYearID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var leaves []domain.StaffLeave
	for rows.Next() {
		l, err := scanLeave(rows)
		if err != nil {
			return nil, err
		}
		leaves = append(leaves, *l)
	}
	return leaves, rows.Err()
}
