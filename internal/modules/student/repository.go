package student

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/ajaypatel01/CampusDesk/internal/domain"
	"github.com/ajaypatel01/CampusDesk/internal/platform/database"
	apperr "github.com/ajaypatel01/CampusDesk/internal/platform/errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

const studentCols = `s.id, s.school_id, s.student_code, s.first_name, s.last_name, s.date_of_birth,
	COALESCE(s.gender,''), COALESCE(s.email,''), COALESCE(s.phone,''), COALESCE(s.address,''),
	s.admission_date, COALESCE(s.caste,''), COALESCE(s.category,''), COALESCE(s.aadhar_number,''),
	COALESCE(s.samagra_id,''), COALESCE(s.pen_number,''), COALESCE(s.apar_id,''),
	COALESCE(s.previous_school,''), COALESCE(s.bank_name,''), COALESCE(s.bank_ifsc,''),
	COALESCE(s.bank_account_number,''), COALESCE(s.bank_holder_name,''), COALESCE(s.bank_branch,''),
	s.status, s.created_at, s.updated_at`

const studentColsSingle = `id, school_id, student_code, first_name, last_name, date_of_birth,
	COALESCE(gender,''), COALESCE(email,''), COALESCE(phone,''), COALESCE(address,''),
	admission_date, COALESCE(caste,''), COALESCE(category,''), COALESCE(aadhar_number,''),
	COALESCE(samagra_id,''), COALESCE(pen_number,''), COALESCE(apar_id,''),
	COALESCE(previous_school,''), COALESCE(bank_name,''), COALESCE(bank_ifsc,''),
	COALESCE(bank_account_number,''), COALESCE(bank_holder_name,''), COALESCE(bank_branch,''),
	status, created_at, updated_at`

var sortColumns = map[string]string{
	"name":           "s.last_name %s, s.first_name %s",
	"student_code":   "s.student_code %s",
	"admission_date": "s.admission_date %s NULLS LAST",
	"class":          "gl.sort_order %s, gl.name %s",
}

type ListFilter struct {
	SchoolID       uuid.UUID
	AcademicYearID uuid.UUID
	Status         string
	Search         string
	Category       string
	GradeLevel     string
	PaymentStatus  string // "paid", "due", "partial"
	SortBy         string
	SortOrder      string
}

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Create(ctx context.Context, s *domain.Student) error {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO students (school_id, student_code, first_name, last_name, date_of_birth,
			gender, email, phone, address, admission_date, caste, category, aadhar_number,
			samagra_id, pen_number, apar_id, previous_school, bank_name, bank_ifsc,
			bank_account_number, bank_holder_name, bank_branch, status)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23)
		RETURNING id, created_at, updated_at`,
		s.SchoolID, s.StudentCode, s.FirstName, s.LastName, s.DateOfBirth,
		s.Gender, s.Email, s.Phone, s.Address, s.AdmissionDate, s.Caste, s.Category,
		s.AadharNumber, s.SamagraID, s.PenNumber, s.AparID, s.PreviousSchool,
		s.BankName, s.BankIFSC, s.BankAccountNumber, s.BankHolderName, s.BankBranch, s.Status,
	)
	if err := row.Scan(&s.ID, &s.CreatedAt, &s.UpdatedAt); err != nil {
		return database.MapError(err)
	}
	return nil
}

func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Student, error) {
	return r.scanOne(ctx, `SELECT `+studentColsSingle+` FROM students WHERE id=$1`, id)
}

func (r *Repository) List(ctx context.Context, f ListFilter, limit, offset int) ([]StudentListItem, int, error) {
	args := []interface{}{f.SchoolID}
	argN := 2

	// Build JOINs for grade level resolution via fee_structures
	joins := " FROM students s"
	joins += " LEFT JOIN student_fee_accounts sfa ON sfa.student_id = s.id"
	if f.AcademicYearID != uuid.Nil {
		joins += fmt.Sprintf(" AND sfa.academic_year_id = $%d", argN)
		args = append(args, f.AcademicYearID)
		argN++
	}
	joins += " LEFT JOIN fee_structures fs ON fs.id = sfa.fee_structure_id"
	joins += " LEFT JOIN grade_levels gl ON gl.id = fs.grade_level_id"
	joins += ` LEFT JOIN LATERAL (
		SELECT COALESCE(SUM(fp.amount), 0) AS total
		FROM fee_payments fp
		WHERE fp.student_fee_account_id = sfa.id AND fp.voided = FALSE
	) paid ON TRUE`

	where := " WHERE s.school_id = $1"

	if f.Status != "" {
		where += fmt.Sprintf(" AND s.status = $%d", argN)
		args = append(args, f.Status)
		argN++
	}
	if f.Search != "" {
		where += fmt.Sprintf(" AND (s.first_name ILIKE $%d OR s.last_name ILIKE $%d OR s.student_code ILIKE $%d)", argN, argN, argN)
		args = append(args, "%"+f.Search+"%")
		argN++
	}
	if f.Category != "" {
		where += fmt.Sprintf(" AND s.category = $%d", argN)
		args = append(args, f.Category)
		argN++
	}
	if f.GradeLevel != "" {
		where += fmt.Sprintf(" AND gl.name = $%d", argN)
		args = append(args, f.GradeLevel)
		argN++
	}
	switch f.PaymentStatus {
	case "paid":
		where += " AND sfa.id IS NOT NULL AND (sfa.tuition_fee - sfa.discount_amount + sfa.van_fee + sfa.previous_year_dues) - paid.total::int <= 0"
	case "due":
		where += " AND sfa.id IS NOT NULL AND (sfa.tuition_fee - sfa.discount_amount + sfa.van_fee + sfa.previous_year_dues) - paid.total::int > 0"
	case "partial":
		where += " AND sfa.id IS NOT NULL AND paid.total > 0 AND (sfa.tuition_fee - sfa.discount_amount + sfa.van_fee + sfa.previous_year_dues) - paid.total::int > 0"
	case "unpaid":
		where += " AND (sfa.id IS NULL OR paid.total = 0)"
	}

	var total int
	countQ := "SELECT COUNT(*)" + joins + where
	if err := r.pool.QueryRow(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	order := buildOrderClause(f.SortBy, f.SortOrder)

	q := fmt.Sprintf(`SELECT %s,
		COALESCE(gl.name, '') AS grade_level_name,
		(CASE WHEN sfa.id IS NOT NULL THEN sfa.tuition_fee - sfa.discount_amount + sfa.van_fee + sfa.previous_year_dues END) AS total_due,
		(CASE WHEN sfa.id IS NOT NULL THEN paid.total::int END) AS total_paid,
		(CASE WHEN sfa.id IS NOT NULL THEN (sfa.tuition_fee - sfa.discount_amount + sfa.van_fee + sfa.previous_year_dues) - paid.total::int END) AS pending_fees,
		COALESCE(sfa.discount_reason, '') AS fee_remarks
		%s %s ORDER BY %s LIMIT $%d OFFSET $%d`,
		studentCols, joins, where, order, argN, argN+1)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var items []StudentListItem
	for rows.Next() {
		item, err := scanListRow(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, *item)
	}
	return items, total, rows.Err()
}

func buildOrderClause(sortBy, sortOrder string) string {
	if sortOrder != "desc" {
		sortOrder = "asc"
	}
	pattern, ok := sortColumns[sortBy]
	if !ok {
		pattern = sortColumns["name"]
	}
	n := strings.Count(pattern, "%s")
	args := make([]interface{}, n)
	for i := range args {
		args[i] = sortOrder
	}
	return fmt.Sprintf(pattern, args...)
}

func (r *Repository) Update(ctx context.Context, s *domain.Student) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE students SET student_code=$2, first_name=$3, last_name=$4, date_of_birth=$5,
			gender=$6, email=$7, phone=$8, address=$9, admission_date=$10, caste=$11,
			category=$12, aadhar_number=$13, samagra_id=$14, pen_number=$15, apar_id=$16,
			previous_school=$17, bank_name=$18, bank_ifsc=$19, bank_account_number=$20,
			bank_holder_name=$21, bank_branch=$22, status=$23, updated_at=NOW()
		WHERE id=$1`,
		s.ID, s.StudentCode, s.FirstName, s.LastName, s.DateOfBirth,
		s.Gender, s.Email, s.Phone, s.Address, s.AdmissionDate, s.Caste, s.Category,
		s.AadharNumber, s.SamagraID, s.PenNumber, s.AparID, s.PreviousSchool,
		s.BankName, s.BankIFSC, s.BankAccountNumber, s.BankHolderName, s.BankBranch, s.Status,
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
	err := row.Scan(
		&s.ID, &s.SchoolID, &s.StudentCode, &s.FirstName, &s.LastName, &s.DateOfBirth,
		&s.Gender, &s.Email, &s.Phone, &s.Address,
		&s.AdmissionDate, &s.Caste, &s.Category, &s.AadharNumber,
		&s.SamagraID, &s.PenNumber, &s.AparID,
		&s.PreviousSchool, &s.BankName, &s.BankIFSC,
		&s.BankAccountNumber, &s.BankHolderName, &s.BankBranch,
		&s.Status, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func scanListRow(row scannable) (*StudentListItem, error) {
	var item StudentListItem
	err := row.Scan(
		&item.ID, &item.SchoolID, &item.StudentCode, &item.FirstName, &item.LastName, &item.DateOfBirth,
		&item.Gender, &item.Email, &item.Phone, &item.Address,
		&item.AdmissionDate, &item.Caste, &item.Category, &item.AadharNumber,
		&item.SamagraID, &item.PenNumber, &item.AparID,
		&item.PreviousSchool, &item.BankName, &item.BankIFSC,
		&item.BankAccountNumber, &item.BankHolderName, &item.BankBranch,
		&item.Status, &item.CreatedAt, &item.UpdatedAt,
		&item.GradeLevelName,
		&item.TotalDue, &item.TotalPaid, &item.PendingFees,
		&item.FeeRemarks,
	)
	if err != nil {
		return nil, err
	}
	return &item, nil
}
