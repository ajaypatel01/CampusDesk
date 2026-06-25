package van

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

// ---- Vans ----

func (r *Repository) CreateVan(ctx context.Context, v *domain.Van) error {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO vans (school_id, van_number, driver_name, driver_phone, capacity, route_name, notes, is_active)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		RETURNING id, created_at, updated_at`,
		v.SchoolID, v.VanNumber, v.DriverName, v.DriverPhone, v.Capacity, v.RouteName, v.Notes, v.IsActive,
	)
	if err := row.Scan(&v.ID, &v.CreatedAt, &v.UpdatedAt); err != nil {
		return database.MapError(err)
	}
	return nil
}

func (r *Repository) GetVanByID(ctx context.Context, id uuid.UUID) (*domain.Van, error) {
	var v domain.Van
	err := r.pool.QueryRow(ctx, `
		SELECT id, school_id, van_number, driver_name, COALESCE(driver_phone,''),
			capacity, COALESCE(route_name,''), COALESCE(notes,''), is_active, created_at, updated_at
		FROM vans WHERE id=$1`, id,
	).Scan(&v.ID, &v.SchoolID, &v.VanNumber, &v.DriverName, &v.DriverPhone,
		&v.Capacity, &v.RouteName, &v.Notes, &v.IsActive, &v.CreatedAt, &v.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperr.ErrNotFound
	}
	return &v, err
}

func (r *Repository) ListVans(ctx context.Context, schoolID uuid.UUID) ([]domain.Van, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, school_id, van_number, driver_name, COALESCE(driver_phone,''),
			capacity, COALESCE(route_name,''), COALESCE(notes,''), is_active, created_at, updated_at
		FROM vans WHERE school_id=$1 ORDER BY van_number`, schoolID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []domain.Van
	for rows.Next() {
		var v domain.Van
		if err := rows.Scan(&v.ID, &v.SchoolID, &v.VanNumber, &v.DriverName, &v.DriverPhone,
			&v.Capacity, &v.RouteName, &v.Notes, &v.IsActive, &v.CreatedAt, &v.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, v)
	}
	return items, rows.Err()
}

func (r *Repository) UpdateVan(ctx context.Context, v *domain.Van) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE vans SET van_number=$2, driver_name=$3, driver_phone=$4, capacity=$5,
			route_name=$6, notes=$7, is_active=$8, updated_at=NOW()
		WHERE id=$1`,
		v.ID, v.VanNumber, v.DriverName, v.DriverPhone, v.Capacity, v.RouteName, v.Notes, v.IsActive,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return apperr.ErrNotFound
	}
	return nil
}

func (r *Repository) DeleteVan(ctx context.Context, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM vans WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return apperr.ErrNotFound
	}
	return nil
}

// ---- Routes ----

func (r *Repository) AddRoute(ctx context.Context, route *domain.VanRoute) error {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO van_routes (van_id, stop_name, stop_order, monthly_fee)
		VALUES ($1,$2,$3,$4)
		RETURNING id, created_at, updated_at`,
		route.VanID, route.StopName, route.StopOrder, route.MonthlyFee,
	)
	if err := row.Scan(&route.ID, &route.CreatedAt, &route.UpdatedAt); err != nil {
		return database.MapError(err)
	}
	return nil
}

func (r *Repository) ListRoutes(ctx context.Context, vanID uuid.UUID) ([]domain.VanRoute, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, van_id, stop_name, stop_order, monthly_fee, created_at, updated_at
		FROM van_routes WHERE van_id=$1 ORDER BY stop_order`, vanID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []domain.VanRoute
	for rows.Next() {
		var rt domain.VanRoute
		if err := rows.Scan(&rt.ID, &rt.VanID, &rt.StopName, &rt.StopOrder, &rt.MonthlyFee, &rt.CreatedAt, &rt.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, rt)
	}
	return items, rows.Err()
}

func (r *Repository) DeleteRoute(ctx context.Context, routeID uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM van_routes WHERE id=$1`, routeID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return apperr.ErrNotFound
	}
	return nil
}

// ---- Assignments ----

func (r *Repository) AssignStudent(ctx context.Context, a *domain.StudentVanAssignment) error {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO student_van_assignments (student_id, van_id, academic_year_id, pickup_stop, assigned_date, is_active)
		VALUES ($1,$2,$3,$4,$5,$6)
		ON CONFLICT (student_id, academic_year_id)
		DO UPDATE SET van_id=$2, pickup_stop=$4, assigned_date=$5, is_active=$6, updated_at=NOW()
		RETURNING id, created_at, updated_at`,
		a.StudentID, a.VanID, a.AcademicYearID, a.PickupStop, a.AssignedDate, a.IsActive,
	)
	if err := row.Scan(&a.ID, &a.CreatedAt, &a.UpdatedAt); err != nil {
		return database.MapError(err)
	}
	return nil
}

func (r *Repository) RemoveAssignment(ctx context.Context, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `UPDATE student_van_assignments SET is_active=FALSE, updated_at=NOW() WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return apperr.ErrNotFound
	}
	return nil
}

type AssignmentDetail struct {
	domain.StudentVanAssignment
	StudentName string `json:"student_name"`
	StudentCode string `json:"student_code"`
}

func (r *Repository) ListAssignments(ctx context.Context, vanID, yearID uuid.UUID) ([]AssignmentDetail, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT a.id, a.student_id, a.van_id, a.academic_year_id, COALESCE(a.pickup_stop,''),
			a.assigned_date, a.is_active, a.created_at, a.updated_at,
			s.first_name || ' ' || s.last_name, s.student_code
		FROM student_van_assignments a
		JOIN students s ON s.id = a.student_id
		WHERE a.van_id=$1 AND a.academic_year_id=$2 AND a.is_active=TRUE
		ORDER BY s.last_name, s.first_name`, vanID, yearID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []AssignmentDetail
	for rows.Next() {
		var d AssignmentDetail
		if err := rows.Scan(&d.ID, &d.StudentID, &d.VanID, &d.AcademicYearID, &d.PickupStop,
			&d.AssignedDate, &d.IsActive, &d.CreatedAt, &d.UpdatedAt,
			&d.StudentName, &d.StudentCode); err != nil {
			return nil, fmt.Errorf("scan assignment: %w", err)
		}
		items = append(items, d)
	}
	return items, rows.Err()
}
