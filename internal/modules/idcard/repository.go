package idcard

import (
	"context"
	"errors"
	"fmt"
	"time"

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

type StudentCardData struct {
	SchoolName    string
	SchoolAddress string
	SchoolPhone   string

	StudentID   uuid.UUID
	StudentName string
	StudentCode string
	ClassName   string
	AcYear      string
	DOB         *time.Time
	Gender      string
	BloodGroup  string
	Phone       string
	Address     string
	PhotoKey    string // S3 object key
}

type TeacherCardData struct {
	SchoolName    string
	SchoolAddress string
	SchoolPhone   string

	UserID      uuid.UUID
	TeacherName string
	EmployeeID  string
	Designation string
	Department  string
	Phone       string
	Email       string
	PhotoKey    string
}

func (r *Repository) GetStudentCardData(ctx context.Context, studentID, yearID uuid.UUID) (*StudentCardData, error) {
	var d StudentCardData
	d.StudentID = studentID

	// Core query — no optional columns (blood_group/photo_url added in migration 005)
	err := r.pool.QueryRow(ctx, `
		SELECT
			sch.name, COALESCE(sch.address,''), COALESCE(sch.phone,''),
			s.first_name || ' ' || s.last_name, s.student_code,
			s.date_of_birth, COALESCE(s.gender,''),
			COALESCE(s.phone,''), COALESCE(s.address,''),
			COALESCE(gl_cs.name, gl_fs.name, ''),
			COALESCE(ay.name,'')
		FROM students s
		JOIN schools sch ON sch.id = s.school_id
		JOIN academic_years ay ON ay.id = $2
		LEFT JOIN enrollments e ON e.student_id = s.id AND e.academic_year_id = $2
		LEFT JOIN class_sections cs ON cs.id = e.class_section_id
		LEFT JOIN grade_levels gl_cs ON gl_cs.id = cs.grade_level_id
		LEFT JOIN student_fee_accounts sfa ON sfa.student_id = s.id AND sfa.academic_year_id = $2
		LEFT JOIN fee_structures fss ON fss.id = sfa.fee_structure_id
		LEFT JOIN grade_levels gl_fs ON gl_fs.id = fss.grade_level_id
		WHERE s.id = $1`, studentID, yearID,
	).Scan(
		&d.SchoolName, &d.SchoolAddress, &d.SchoolPhone,
		&d.StudentName, &d.StudentCode,
		&d.DOB, &d.Gender,
		&d.Phone, &d.Address,
		&d.ClassName, &d.AcYear,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperr.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get student card data: %w", err)
	}

	// Optional extended fields (migration 005) — ignore if columns not yet added
	_ = r.pool.QueryRow(ctx, `
		SELECT COALESCE(blood_group,''), COALESCE(photo_url,'')
		FROM students WHERE id = $1`, studentID,
	).Scan(&d.BloodGroup, &d.PhotoKey)

	return &d, nil
}

func (r *Repository) GetTeacherCardData(ctx context.Context, userID uuid.UUID) (*TeacherCardData, error) {
	var d TeacherCardData
	d.UserID = userID

	// Core query — role column always present
	err := r.pool.QueryRow(ctx, `
		SELECT
			COALESCE(sch.name,''), COALESCE(sch.address,''), COALESCE(sch.phone,''),
			u.first_name || ' ' || u.last_name,
			u.role, COALESCE(sp.phone,''), u.email,
			COALESCE(sp.designation, u.role)
		FROM users u
		LEFT JOIN schools sch ON sch.id = u.school_id
		LEFT JOIN staff_profiles sp ON sp.user_id = u.id
		WHERE u.id = $1`, userID,
	).Scan(
		&d.SchoolName, &d.SchoolAddress, &d.SchoolPhone,
		&d.TeacherName, &d.Department, &d.Phone, &d.Email,
		&d.Designation,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperr.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get teacher card data: %w", err)
	}

	// Optional extended fields (migration 005) — ignore if columns not yet added
	_ = r.pool.QueryRow(ctx, `
		SELECT COALESCE(employee_id,''), COALESCE(department,''), COALESCE(photo_url,'')
		FROM users WHERE id = $1`, userID,
	).Scan(&d.EmployeeID, &d.Department, &d.PhotoKey)

	return &d, nil
}
