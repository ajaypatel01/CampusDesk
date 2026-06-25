package rte

import (
	"context"
	"errors"

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

func (r *Repository) UpsertQuota(ctx context.Context, q *domain.RTEQuota) error {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO rte_quotas (school_id, academic_year_id, grade_level_id, total_seats, govt_reimbursement_per_student, notes)
		VALUES ($1,$2,$3,$4,$5,$6)
		ON CONFLICT (school_id, academic_year_id, grade_level_id)
		DO UPDATE SET total_seats=$4, govt_reimbursement_per_student=$5, notes=$6, updated_at=NOW()
		RETURNING id, created_at, updated_at`,
		q.SchoolID, q.AcademicYearID, q.GradeLevelID, q.TotalSeats, q.GovtReimbursementPerStudent, q.Notes,
	)
	if err := row.Scan(&q.ID, &q.CreatedAt, &q.UpdatedAt); err != nil {
		return database.MapError(err)
	}
	return nil
}

func (r *Repository) GetQuotaByID(ctx context.Context, id uuid.UUID) (*domain.RTEQuota, error) {
	var q domain.RTEQuota
	err := r.pool.QueryRow(ctx, `
		SELECT id, school_id, academic_year_id, grade_level_id, total_seats,
			govt_reimbursement_per_student, COALESCE(notes,''), created_at, updated_at
		FROM rte_quotas WHERE id=$1`, id,
	).Scan(&q.ID, &q.SchoolID, &q.AcademicYearID, &q.GradeLevelID, &q.TotalSeats,
		&q.GovtReimbursementPerStudent, &q.Notes, &q.CreatedAt, &q.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperr.ErrNotFound
	}
	return &q, err
}

func (r *Repository) ListQuotas(ctx context.Context, schoolID, yearID uuid.UUID) ([]RTEQuotaDetail, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT q.id, q.school_id, q.academic_year_id, q.grade_level_id, q.total_seats,
			q.govt_reimbursement_per_student, COALESCE(q.notes,''), q.created_at, q.updated_at,
			gl.name,
			COUNT(sfa.id) FILTER (WHERE sfa.is_rte = TRUE) AS utilized
		FROM rte_quotas q
		JOIN grade_levels gl ON gl.id = q.grade_level_id
		LEFT JOIN fee_structures fs ON fs.grade_level_id = q.grade_level_id AND fs.academic_year_id = q.academic_year_id
		LEFT JOIN student_fee_accounts sfa ON sfa.fee_structure_id = fs.id AND sfa.is_rte = TRUE
		WHERE q.school_id=$1 AND q.academic_year_id=$2
		GROUP BY q.id, gl.name
		ORDER BY gl.sort_order`, schoolID, yearID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []RTEQuotaDetail
	for rows.Next() {
		var d RTEQuotaDetail
		if err := rows.Scan(&d.ID, &d.SchoolID, &d.AcademicYearID, &d.GradeLevelID, &d.TotalSeats,
			&d.GovtReimbursementPerStudent, &d.Notes, &d.CreatedAt, &d.UpdatedAt,
			&d.GradeLevelName, &d.UtilizedSeats); err != nil {
			return nil, err
		}
		d.AvailableSeats = d.TotalSeats - d.UtilizedSeats
		d.TotalReimbursement = d.UtilizedSeats * d.GovtReimbursementPerStudent
		items = append(items, d)
	}
	return items, rows.Err()
}

func (r *Repository) DeleteQuota(ctx context.Context, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM rte_quotas WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return apperr.ErrNotFound
	}
	return nil
}

type RTEStudent struct {
	StudentID      uuid.UUID `json:"student_id"`
	StudentName    string    `json:"student_name"`
	StudentCode    string    `json:"student_code"`
	GradeLevelName string    `json:"grade_level_name"`
	FeeAccountID   uuid.UUID `json:"fee_account_id"`
}

func (r *Repository) ListRTEStudents(ctx context.Context, schoolID, yearID uuid.UUID) ([]RTEStudent, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT s.id, s.first_name || ' ' || s.last_name, s.student_code,
			COALESCE(gl.name,''), sfa.id
		FROM student_fee_accounts sfa
		JOIN students s ON s.id = sfa.student_id
		JOIN fee_structures fs ON fs.id = sfa.fee_structure_id
		LEFT JOIN grade_levels gl ON gl.id = fs.grade_level_id
		WHERE sfa.school_id=$1 AND sfa.academic_year_id=$2 AND sfa.is_rte=TRUE
		ORDER BY gl.sort_order, s.last_name`, schoolID, yearID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []RTEStudent
	for rows.Next() {
		var st RTEStudent
		if err := rows.Scan(&st.StudentID, &st.StudentName, &st.StudentCode, &st.GradeLevelName, &st.FeeAccountID); err != nil {
			return nil, err
		}
		items = append(items, st)
	}
	return items, rows.Err()
}

type RTESummary struct {
	SchoolID         uuid.UUID        `json:"school_id"`
	AcademicYearID   uuid.UUID        `json:"academic_year_id"`
	TotalSeats       int              `json:"total_seats"`
	UtilizedSeats    int              `json:"utilized_seats"`
	AvailableSeats   int              `json:"available_seats"`
	TotalReimbursement int            `json:"total_reimbursement"`
	ByGrade          []RTEQuotaDetail `json:"by_grade"`
}

func (r *Repository) GetSummary(ctx context.Context, schoolID, yearID uuid.UUID) (*RTESummary, error) {
	quotas, err := r.ListQuotas(ctx, schoolID, yearID)
	if err != nil {
		return nil, err
	}
	summary := &RTESummary{SchoolID: schoolID, AcademicYearID: yearID, ByGrade: quotas}
	for _, q := range quotas {
		summary.TotalSeats += q.TotalSeats
		summary.UtilizedSeats += q.UtilizedSeats
		summary.TotalReimbursement += q.TotalReimbursement
	}
	summary.AvailableSeats = summary.TotalSeats - summary.UtilizedSeats
	return summary, nil
}
