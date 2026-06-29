package staff

import (
	"context"
	"errors"
	"fmt"

	"github.com/ajaypatel01/CampusDesk/internal/domain"
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

const staffSelect = `
	SELECT
		u.id, u.school_id, u.email, u.first_name, u.last_name, u.role, u.is_active, u.created_at, u.updated_at,
		sp.id, sp.user_id, sp.guardian_name, sp.aadhar_number, sp.education_qualification,
		sp.professional_qualification, sp.designation, sp.salary,
		sp.bank_name, sp.bank_ifsc, sp.bank_branch, sp.bank_account_number, sp.bank_account_holder,
		sp.phone, sp.staff_type, sp.created_at, sp.updated_at
	FROM users u
	LEFT JOIN staff_profiles sp ON sp.user_id = u.id`

func (r *Repository) List(ctx context.Context, schoolID *uuid.UUID, limit, offset int) ([]domain.StaffMember, int, error) {
	var total int
	var rows pgx.Rows
	var err error

	if schoolID != nil {
		if err = r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM users WHERE school_id=$1 AND role IN ('teacher','school_admin','registrar','super_admin')`, *schoolID).Scan(&total); err != nil {
			return nil, 0, err
		}
		rows, err = r.pool.Query(ctx,
			staffSelect+` WHERE u.school_id=$1 AND u.role IN ('teacher','school_admin','registrar','super_admin') ORDER BY sp.designation NULLS LAST, u.last_name, u.first_name LIMIT $2 OFFSET $3`,
			*schoolID, limit, offset,
		)
	} else {
		if err = r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM users WHERE role IN ('teacher','school_admin','registrar','super_admin')`).Scan(&total); err != nil {
			return nil, 0, err
		}
		rows, err = r.pool.Query(ctx,
			staffSelect+` WHERE u.role IN ('teacher','school_admin','registrar','super_admin') ORDER BY sp.designation NULLS LAST, u.last_name, u.first_name LIMIT $1 OFFSET $2`,
			limit, offset,
		)
	}
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var members []domain.StaffMember
	for rows.Next() {
		m, err := scanMember(rows)
		if err != nil {
			return nil, 0, err
		}
		members = append(members, *m)
	}
	return members, total, rows.Err()
}

func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*domain.StaffMember, error) {
	row := r.pool.QueryRow(ctx, staffSelect+` WHERE u.id=$1`, id)
	m, err := scanMember(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperr.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan staff member: %w", err)
	}
	return m, nil
}

func (r *Repository) UpsertProfile(ctx context.Context, p *domain.StaffProfile) error {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO staff_profiles (user_id, guardian_name, aadhar_number, education_qualification,
			professional_qualification, designation, salary, bank_name, bank_ifsc, bank_branch,
			bank_account_number, bank_account_holder, phone, staff_type)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
		ON CONFLICT (user_id) DO UPDATE SET
			guardian_name=$2, aadhar_number=$3, education_qualification=$4,
			professional_qualification=$5, designation=$6, salary=$7, bank_name=$8,
			bank_ifsc=$9, bank_branch=$10, bank_account_number=$11,
			bank_account_holder=$12, phone=$13, staff_type=$14, updated_at=NOW()
		RETURNING id, created_at, updated_at`,
		p.UserID, p.GuardianName, p.AadharNumber, p.EducationQualification,
		p.ProfessionalQualification, p.Designation, p.Salary, p.BankName,
		p.BankIFSC, p.BankBranch, p.BankAccountNumber, p.BankAccountHolder, p.Phone, p.StaffType,
	)
	return row.Scan(&p.ID, &p.CreatedAt, &p.UpdatedAt)
}

type scannable interface {
	Scan(dest ...interface{}) error
}

func scanMember(row scannable) (*domain.StaffMember, error) {
	var m domain.StaffMember
	var p domain.StaffProfile
	var profileID *uuid.UUID
	var profileUserID *uuid.UUID

	err := row.Scan(
		&m.ID, &m.SchoolID, &m.Email, &m.FirstName, &m.LastName, &m.Role, &m.IsActive, &m.CreatedAt, &m.UpdatedAt,
		&profileID, &profileUserID, &p.GuardianName, &p.AadharNumber, &p.EducationQualification,
		&p.ProfessionalQualification, &p.Designation, &p.Salary,
		&p.BankName, &p.BankIFSC, &p.BankBranch, &p.BankAccountNumber, &p.BankAccountHolder,
		&p.Phone, &p.StaffType, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if profileID != nil {
		p.ID = *profileID
		p.UserID = m.ID
		m.Profile = &p
	}
	return &m, nil
}
