package tcvoucher

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

// ---- TC Records ----

type TCRecord struct {
	ID              uuid.UUID  `json:"id"`
	SchoolID        uuid.UUID  `json:"school_id"`
	ScholarNumber   *string    `json:"scholar_number,omitempty"`
	StudentName     string     `json:"student_name"`
	FatherName      *string    `json:"father_name,omitempty"`
	MotherName      *string    `json:"mother_name,omitempty"`
	DOB             *time.Time `json:"dob,omitempty"`
	Caste           *string    `json:"caste,omitempty"`
	Category        *string    `json:"category,omitempty"`
	DateOfAdmission *time.Time `json:"date_of_admission,omitempty"`
	ApplicationDate *time.Time `json:"application_date,omitempty"`
	IssueDate       *time.Time `json:"issue_date,omitempty"`
	ClassPassed     *string    `json:"class_passed,omitempty"`
	PENNumber       *string    `json:"pen_number,omitempty"`
	APARID          *string    `json:"apar_id,omitempty"`
	SamagraID       *string    `json:"samagra_id,omitempty"`
	NewSchool       *string    `json:"new_school,omitempty"`
	DICECode        *string    `json:"dice_code,omitempty"`
	Remark          *string    `json:"remark,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
}

func (r *Repository) ListTCRecords(ctx context.Context, schoolID uuid.UUID, limit, offset int) ([]TCRecord, int, error) {
	var total int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM tc_records WHERE school_id=$1`, schoolID).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id,school_id,scholar_number,student_name,father_name,mother_name,dob,
			caste,category,date_of_admission,application_date,issue_date,class_passed,
			pen_number,apar_id,samagra_id,new_school,dice_code,remark,created_at
		FROM tc_records WHERE school_id=$1 ORDER BY issue_date DESC NULLS LAST, created_at DESC
		LIMIT $2 OFFSET $3`, schoolID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var items []TCRecord
	for rows.Next() {
		var t TCRecord
		if err := rows.Scan(&t.ID, &t.SchoolID, &t.ScholarNumber, &t.StudentName, &t.FatherName, &t.MotherName,
			&t.DOB, &t.Caste, &t.Category, &t.DateOfAdmission, &t.ApplicationDate, &t.IssueDate, &t.ClassPassed,
			&t.PENNumber, &t.APARID, &t.SamagraID, &t.NewSchool, &t.DICECode, &t.Remark, &t.CreatedAt); err != nil {
			return nil, 0, err
		}
		items = append(items, t)
	}
	return items, total, rows.Err()
}

func (r *Repository) CreateTCRecord(ctx context.Context, t *TCRecord) error {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO tc_records (school_id,scholar_number,student_name,father_name,mother_name,dob,
			caste,category,date_of_admission,application_date,issue_date,class_passed,
			pen_number,apar_id,samagra_id,new_school,dice_code,remark)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18)
		RETURNING id,created_at`,
		t.SchoolID, t.ScholarNumber, t.StudentName, t.FatherName, t.MotherName, t.DOB,
		t.Caste, t.Category, t.DateOfAdmission, t.ApplicationDate, t.IssueDate, t.ClassPassed,
		t.PENNumber, t.APARID, t.SamagraID, t.NewSchool, t.DICECode, t.Remark,
	)
	return row.Scan(&t.ID, &t.CreatedAt)
}

func (r *Repository) GetTCRecord(ctx context.Context, id uuid.UUID) (*TCRecord, error) {
	var t TCRecord
	err := r.pool.QueryRow(ctx, `
		SELECT id,school_id,scholar_number,student_name,father_name,mother_name,dob,
			caste,category,date_of_admission,application_date,issue_date,class_passed,
			pen_number,apar_id,samagra_id,new_school,dice_code,remark,created_at
		FROM tc_records WHERE id=$1`, id,
	).Scan(&t.ID, &t.SchoolID, &t.ScholarNumber, &t.StudentName, &t.FatherName, &t.MotherName,
		&t.DOB, &t.Caste, &t.Category, &t.DateOfAdmission, &t.ApplicationDate, &t.IssueDate, &t.ClassPassed,
		&t.PENNumber, &t.APARID, &t.SamagraID, &t.NewSchool, &t.DICECode, &t.Remark, &t.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperr.ErrNotFound
	}
	return &t, err
}

// ---- Vouchers ----

type Voucher struct {
	ID            uuid.UUID `json:"id"`
	SchoolID      uuid.UUID `json:"school_id"`
	Date          time.Time `json:"date"`
	AccountName   string    `json:"account_name"`
	Payee         *string   `json:"payee,omitempty"`
	Amount        float64   `json:"amount"`
	Description   *string   `json:"description,omitempty"`
	ModeOfPayment *string   `json:"mode_of_payment,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

func (r *Repository) ListVouchers(ctx context.Context, schoolID uuid.UUID, from, to *time.Time, limit, offset int) ([]Voucher, int, error) {
	args := []interface{}{schoolID}
	where := "WHERE school_id=$1"
	n := 2
	if from != nil {
		where += fmt.Sprintf(" AND date >= $%d", n)
		args = append(args, *from)
		n++
	}
	if to != nil {
		where += fmt.Sprintf(" AND date <= $%d", n)
		args = append(args, *to)
		n++
	}

	var total int
	if err := r.pool.QueryRow(ctx, fmt.Sprintf(`SELECT COUNT(*) FROM vouchers %s`, where), args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args = append(args, limit, offset)
	rows, err := r.pool.Query(ctx, fmt.Sprintf(`
		SELECT id,school_id,date,account_name,payee,amount,description,mode_of_payment,created_at
		FROM vouchers %s ORDER BY date DESC, created_at DESC LIMIT $%d OFFSET $%d`, where, n, n+1), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var items []Voucher
	for rows.Next() {
		var v Voucher
		if err := rows.Scan(&v.ID, &v.SchoolID, &v.Date, &v.AccountName, &v.Payee, &v.Amount, &v.Description, &v.ModeOfPayment, &v.CreatedAt); err != nil {
			return nil, 0, err
		}
		items = append(items, v)
	}
	return items, total, rows.Err()
}

func (r *Repository) CreateVoucher(ctx context.Context, v *Voucher) error {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO vouchers (school_id,date,account_name,payee,amount,description,mode_of_payment)
		VALUES ($1,$2,$3,$4,$5,$6,$7) RETURNING id,created_at`,
		v.SchoolID, v.Date, v.AccountName, v.Payee, v.Amount, v.Description, v.ModeOfPayment,
	)
	return row.Scan(&v.ID, &v.CreatedAt)
}
