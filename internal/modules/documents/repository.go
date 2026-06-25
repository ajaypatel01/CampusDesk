package documents

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

func (r *Repository) GetBonafideData(ctx context.Context, studentID, yearID uuid.UUID) (*BonafideData, error) {
	var d BonafideData
	err := r.pool.QueryRow(ctx, `
		SELECT
			sch.name, COALESCE(sch.address,''), COALESCE(sch.phone,''), COALESCE(sch.email,''),
			s.first_name || ' ' || s.last_name, s.student_code,
			s.date_of_birth, COALESCE(s.gender,''), COALESCE(s.category,''), COALESCE(s.caste,''),
			s.admission_date,
			COALESCE(gl_cs.name, gl_fs.name, ''), COALESCE(ay.name,''),
			COALESCE(g.first_name || ' ' || g.last_name, ''), COALESCE(g.relation,'')
		FROM students s
		JOIN schools sch ON sch.id = s.school_id
		JOIN academic_years ay ON ay.id = $2
		LEFT JOIN enrollments e ON e.student_id = s.id AND e.academic_year_id = $2
		LEFT JOIN class_sections cs ON cs.id = e.class_section_id
		LEFT JOIN grade_levels gl_cs ON gl_cs.id = cs.grade_level_id
		LEFT JOIN student_fee_accounts sfa ON sfa.student_id = s.id AND sfa.academic_year_id = $2
		LEFT JOIN fee_structures fss ON fss.id = sfa.fee_structure_id
		LEFT JOIN grade_levels gl_fs ON gl_fs.id = fss.grade_level_id
		LEFT JOIN student_guardians sg ON sg.student_id = s.id AND sg.is_primary = TRUE
		LEFT JOIN guardians g ON g.id = sg.guardian_id
		WHERE s.id = $1`, studentID, yearID,
	).Scan(
		&d.SchoolName, &d.SchoolAddress, &d.SchoolPhone, &d.SchoolEmail,
		&d.StudentName, &d.StudentCode,
		&d.DOB, &d.Gender, &d.Category, &d.Caste,
		&d.AdmissionDate,
		&d.ClassName, &d.AcademicYear,
		&d.GuardianName, &d.GuardianRelation,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperr.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get bonafide data: %w", err)
	}
	d.IssueDate = time.Now()
	return &d, nil
}

func (r *Repository) GetTCData(ctx context.Context, studentID uuid.UUID) (*TCData, error) {
	var d TCData
	err := r.pool.QueryRow(ctx, `
		WITH first_enroll AS (
			SELECT e.id, cs.grade_level_id
			FROM enrollments e
			LEFT JOIN class_sections cs ON cs.id = e.class_section_id
			JOIN academic_years ay ON ay.id = e.academic_year_id
			WHERE e.student_id = $1
			ORDER BY ay.start_date ASC
			LIMIT 1
		),
		last_enroll AS (
			SELECT e.id, cs.grade_level_id, ay.name AS year_name
			FROM enrollments e
			LEFT JOIN class_sections cs ON cs.id = e.class_section_id
			JOIN academic_years ay ON ay.id = e.academic_year_id
			WHERE e.student_id = $1
			ORDER BY ay.start_date DESC
			LIMIT 1
		),
		fee_grade AS (
			SELECT gl.id, gl.name
			FROM student_fee_accounts sfa
			JOIN fee_structures fs ON fs.id = sfa.fee_structure_id
			JOIN grade_levels gl ON gl.id = fs.grade_level_id
			JOIN academic_years ay ON ay.id = sfa.academic_year_id
			WHERE sfa.student_id = $1
			ORDER BY ay.start_date DESC
			LIMIT 1
		),
		first_fee_grade AS (
			SELECT gl.id, gl.name
			FROM student_fee_accounts sfa
			JOIN fee_structures fs ON fs.id = sfa.fee_structure_id
			JOIN grade_levels gl ON gl.id = fs.grade_level_id
			JOIN academic_years ay ON ay.id = sfa.academic_year_id
			WHERE sfa.student_id = $1
			ORDER BY ay.start_date ASC
			LIMIT 1
		),
		balance AS (
			SELECT COALESCE(
				SUM(sfa.tuition_fee - sfa.discount_amount + sfa.van_fee + sfa.previous_year_dues) -
				COALESCE(SUM(fp.amount) FILTER (WHERE fp.voided = FALSE), 0), 0
			) AS remaining
			FROM student_fee_accounts sfa
			LEFT JOIN fee_payments fp ON fp.student_fee_account_id = sfa.id
			WHERE sfa.student_id = $1
		)
		SELECT
			sch.name, COALESCE(sch.address,''), COALESCE(sch.phone,''), COALESCE(sch.email,''),
			s.first_name || ' ' || s.last_name, s.student_code,
			s.date_of_birth, COALESCE(s.gender,''), COALESCE(s.category,''), COALESCE(s.caste,''),
			s.admission_date, COALESCE(s.aadhar_number,''), COALESCE(s.previous_school,''),
			COALESCE(gl_first.name, ffg.name, ''),
			COALESCE(gl_last.name, fg.name, ''),
			COALESCE(le.year_name, ''),
			COALESCE(g.first_name || ' ' || g.last_name, ''), COALESCE(g.relation,''),
			COALESCE(balance.remaining, 0)
		FROM students s
		JOIN schools sch ON sch.id = s.school_id
		LEFT JOIN first_enroll fe ON TRUE
		LEFT JOIN grade_levels gl_first ON gl_first.id = fe.grade_level_id
		LEFT JOIN last_enroll le ON TRUE
		LEFT JOIN grade_levels gl_last ON gl_last.id = le.grade_level_id
		LEFT JOIN fee_grade fg ON TRUE
		LEFT JOIN first_fee_grade ffg ON TRUE
		LEFT JOIN student_guardians sg ON sg.student_id = s.id AND sg.is_primary = TRUE
		LEFT JOIN guardians g ON g.id = sg.guardian_id
		LEFT JOIN balance ON TRUE
		WHERE s.id = $1`, studentID,
	).Scan(
		&d.SchoolName, &d.SchoolAddress, &d.SchoolPhone, &d.SchoolEmail,
		&d.StudentName, &d.StudentCode,
		&d.DOB, &d.Gender, &d.Category, &d.Caste,
		&d.AdmissionDate, &d.AadharNumber, &d.PreviousSchool,
		&d.AdmittedClass, &d.LastClass, &d.LastAcademicYear,
		&d.GuardianName, &d.GuardianRelation,
		&d.OutstandingFees,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperr.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get tc data: %w", err)
	}
	d.FeeCleared = d.OutstandingFees <= 0
	d.IssueDate = time.Now()
	return &d, nil
}

func (r *Repository) GetSchoolName(ctx context.Context, schoolID uuid.UUID) (name, address, phone, email string, err error) {
	err = r.pool.QueryRow(ctx, `
		SELECT name, COALESCE(address,''), COALESCE(phone,''), COALESCE(email,'')
		FROM schools WHERE id = $1`, schoolID,
	).Scan(&name, &address, &phone, &email)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", "", "", apperr.ErrNotFound
	}
	return
}
