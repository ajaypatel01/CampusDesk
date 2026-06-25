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

	err := r.pool.QueryRow(ctx, `
		SELECT
			sch.name, COALESCE(sch.address,''), COALESCE(sch.phone,''),
			s.first_name || ' ' || s.last_name, s.student_code,
			s.date_of_birth, COALESCE(s.gender,''), COALESCE(s.blood_group,''),
			COALESCE(s.phone,''), COALESCE(s.address,''),
			COALESCE(s.photo_url,''),
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
		&d.DOB, &d.Gender, &d.BloodGroup,
		&d.Phone, &d.Address, &d.PhotoKey,
		&d.ClassName, &d.AcYear,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperr.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get student card data: %w", err)
	}
	return &d, nil
}

func (r *Repository) GetTeacherCardData(ctx context.Context, userID uuid.UUID) (*TeacherCardData, error) {
	var d TeacherCardData
	d.UserID = userID

	err := r.pool.QueryRow(ctx, `
		SELECT
			COALESCE(sch.name,''), COALESCE(sch.address,''), COALESCE(sch.phone,''),
			u.first_name || ' ' || u.last_name,
			COALESCE(u.employee_id,''), u.role,
			COALESCE(u.department,''), COALESCE(u.phone,''),
			u.email, COALESCE(u.photo_url,'')
		FROM users u
		LEFT JOIN schools sch ON sch.id = u.school_id
		WHERE u.id = $1`, userID,
	).Scan(
		&d.SchoolName, &d.SchoolAddress, &d.SchoolPhone,
		&d.TeacherName, &d.EmployeeID, &d.Designation,
		&d.Department, &d.Phone, &d.Email, &d.PhotoKey,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperr.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get teacher card data: %w", err)
	}
	return &d, nil
}
