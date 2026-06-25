package fee

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

// ---- Fee Structures ----

func (r *Repository) CreateFeeStructure(ctx context.Context, fs *domain.FeeStructure, plans []domain.FeeInstallmentPlan) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	row := tx.QueryRow(ctx, `
		INSERT INTO fee_structures (school_id, academic_year_id, grade_level_id, tuition_fee_annual, num_installments, van_fee_annual)
		VALUES ($1,$2,$3,$4,$5,$6) RETURNING id, created_at, updated_at`,
		fs.SchoolID, fs.AcademicYearID, fs.GradeLevelID, fs.TuitionFeeAnnual, fs.NumInstallments, fs.VanFeeAnnual,
	)
	if err := row.Scan(&fs.ID, &fs.CreatedAt, &fs.UpdatedAt); err != nil {
		return database.MapError(err)
	}

	for i := range plans {
		plans[i].FeeStructureID = fs.ID
		row := tx.QueryRow(ctx, `
			INSERT INTO fee_installment_plans (fee_structure_id, installment_number, label, amount, due_date)
			VALUES ($1,$2,$3,$4,$5) RETURNING id, created_at, updated_at`,
			plans[i].FeeStructureID, plans[i].InstallmentNumber, plans[i].Label, plans[i].Amount, plans[i].DueDate,
		)
		if err := row.Scan(&plans[i].ID, &plans[i].CreatedAt, &plans[i].UpdatedAt); err != nil {
			return database.MapError(err)
		}
	}

	return tx.Commit(ctx)
}

func (r *Repository) GetFeeStructureByID(ctx context.Context, id uuid.UUID) (*domain.FeeStructure, error) {
	var fs domain.FeeStructure
	err := r.pool.QueryRow(ctx, `
		SELECT id, school_id, academic_year_id, grade_level_id, tuition_fee_annual, num_installments, van_fee_annual, created_at, updated_at
		FROM fee_structures WHERE id=$1`, id,
	).Scan(&fs.ID, &fs.SchoolID, &fs.AcademicYearID, &fs.GradeLevelID, &fs.TuitionFeeAnnual, &fs.NumInstallments, &fs.VanFeeAnnual, &fs.CreatedAt, &fs.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperr.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get fee structure: %w", err)
	}
	return &fs, nil
}

func (r *Repository) ListFeeStructures(ctx context.Context, schoolID, yearID uuid.UUID) ([]domain.FeeStructure, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, school_id, academic_year_id, grade_level_id, tuition_fee_annual, num_installments, van_fee_annual, created_at, updated_at
		FROM fee_structures WHERE school_id=$1 AND academic_year_id=$2
		ORDER BY tuition_fee_annual`, schoolID, yearID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []domain.FeeStructure
	for rows.Next() {
		var fs domain.FeeStructure
		if err := rows.Scan(&fs.ID, &fs.SchoolID, &fs.AcademicYearID, &fs.GradeLevelID, &fs.TuitionFeeAnnual, &fs.NumInstallments, &fs.VanFeeAnnual, &fs.CreatedAt, &fs.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, fs)
	}
	return items, rows.Err()
}

func (r *Repository) UpdateFeeStructure(ctx context.Context, fs *domain.FeeStructure, plans []domain.FeeInstallmentPlan) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	tag, err := tx.Exec(ctx, `
		UPDATE fee_structures SET tuition_fee_annual=$2, num_installments=$3, van_fee_annual=$4, updated_at=NOW() WHERE id=$1`,
		fs.ID, fs.TuitionFeeAnnual, fs.NumInstallments, fs.VanFeeAnnual,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return apperr.ErrNotFound
	}

	if _, err := tx.Exec(ctx, `DELETE FROM fee_installment_plans WHERE fee_structure_id=$1`, fs.ID); err != nil {
		return err
	}

	for i := range plans {
		plans[i].FeeStructureID = fs.ID
		row := tx.QueryRow(ctx, `
			INSERT INTO fee_installment_plans (fee_structure_id, installment_number, label, amount, due_date)
			VALUES ($1,$2,$3,$4,$5) RETURNING id, created_at, updated_at`,
			plans[i].FeeStructureID, plans[i].InstallmentNumber, plans[i].Label, plans[i].Amount, plans[i].DueDate,
		)
		if err := row.Scan(&plans[i].ID, &plans[i].CreatedAt, &plans[i].UpdatedAt); err != nil {
			return database.MapError(err)
		}
	}

	return tx.Commit(ctx)
}

func (r *Repository) ListInstallmentPlans(ctx context.Context, feeStructureID uuid.UUID) ([]domain.FeeInstallmentPlan, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, fee_structure_id, installment_number, label, amount, due_date, created_at, updated_at
		FROM fee_installment_plans WHERE fee_structure_id=$1 ORDER BY installment_number`, feeStructureID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []domain.FeeInstallmentPlan
	for rows.Next() {
		var ip domain.FeeInstallmentPlan
		if err := rows.Scan(&ip.ID, &ip.FeeStructureID, &ip.InstallmentNumber, &ip.Label, &ip.Amount, &ip.DueDate, &ip.CreatedAt, &ip.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, ip)
	}
	return items, rows.Err()
}

// ---- Student Fee Accounts ----

func (r *Repository) CreateFeeAccount(ctx context.Context, fa *domain.StudentFeeAccount) error {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO student_fee_accounts (student_id, school_id, academic_year_id, fee_structure_id,
			tuition_fee, discount_amount, discount_reason, previous_year_dues, van_fee, is_rte)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) RETURNING id, created_at, updated_at`,
		fa.StudentID, fa.SchoolID, fa.AcademicYearID, fa.FeeStructureID,
		fa.TuitionFee, fa.DiscountAmount, fa.DiscountReason, fa.PreviousYearDues, fa.VanFee, fa.IsRTE,
	)
	if err := row.Scan(&fa.ID, &fa.CreatedAt, &fa.UpdatedAt); err != nil {
		return database.MapError(err)
	}
	return nil
}

func (r *Repository) GetFeeAccountByID(ctx context.Context, id uuid.UUID) (*domain.StudentFeeAccount, error) {
	var fa domain.StudentFeeAccount
	err := r.pool.QueryRow(ctx, `
		SELECT id, student_id, school_id, academic_year_id, fee_structure_id,
			tuition_fee, discount_amount, COALESCE(discount_reason,''), previous_year_dues, van_fee, is_rte,
			created_at, updated_at
		FROM student_fee_accounts WHERE id=$1`, id,
	).Scan(&fa.ID, &fa.StudentID, &fa.SchoolID, &fa.AcademicYearID, &fa.FeeStructureID,
		&fa.TuitionFee, &fa.DiscountAmount, &fa.DiscountReason, &fa.PreviousYearDues, &fa.VanFee, &fa.IsRTE,
		&fa.CreatedAt, &fa.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperr.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get fee account: %w", err)
	}
	return &fa, nil
}

func (r *Repository) ListFeeAccounts(ctx context.Context, f FeeAccountFilter, limit, offset int) ([]FeeAccountSummary, int, error) {
	args := []interface{}{f.SchoolID, f.AcademicYearID}
	where := "WHERE sfa.school_id = $1 AND sfa.academic_year_id = $2"
	argN := 3

	if f.Search != "" {
		where += fmt.Sprintf(" AND (s.first_name ILIKE $%d OR s.last_name ILIKE $%d OR s.student_code ILIKE $%d)", argN, argN, argN)
		args = append(args, "%"+f.Search+"%")
		argN++
	}
	if f.GradeLevel != "" {
		where += fmt.Sprintf(" AND gl.name = $%d", argN)
		args = append(args, f.GradeLevel)
		argN++
	}

	having := ""
	switch f.PaymentStatus {
	case "paid":
		having = " HAVING (sfa.tuition_fee - sfa.discount_amount + sfa.van_fee + sfa.previous_year_dues) - COALESCE(SUM(fp.amount) FILTER (WHERE fp.voided = FALSE), 0) <= 0"
	case "due":
		having = " HAVING (sfa.tuition_fee - sfa.discount_amount + sfa.van_fee + sfa.previous_year_dues) - COALESCE(SUM(fp.amount) FILTER (WHERE fp.voided = FALSE), 0) > 0"
	case "partial":
		having = " HAVING COALESCE(SUM(fp.amount) FILTER (WHERE fp.voided = FALSE), 0) > 0 AND (sfa.tuition_fee - sfa.discount_amount + sfa.van_fee + sfa.previous_year_dues) - COALESCE(SUM(fp.amount) FILTER (WHERE fp.voided = FALSE), 0) > 0"
	}

	countBase := `FROM student_fee_accounts sfa
		JOIN students s ON s.id = sfa.student_id
		JOIN fee_structures fs ON fs.id = sfa.fee_structure_id
		JOIN grade_levels gl ON gl.id = fs.grade_level_id
		LEFT JOIN fee_payments fp ON fp.student_fee_account_id = sfa.id `

	var total int
	countQ := `SELECT COUNT(*) FROM (SELECT sfa.id ` + countBase + where +
		` GROUP BY sfa.id, s.first_name, s.last_name, s.student_code, gl.name, sfa.tuition_fee, sfa.discount_amount, sfa.van_fee, sfa.previous_year_dues` + having + `) sub`
	if err := r.pool.QueryRow(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	q := fmt.Sprintf(`
		SELECT sfa.id, sfa.student_id,
			s.first_name || ' ' || s.last_name AS student_name,
			s.student_code,
			gl.name AS grade_level_name,
			sfa.tuition_fee, sfa.discount_amount, sfa.van_fee,
			sfa.previous_year_dues, sfa.is_rte,
			(sfa.tuition_fee - sfa.discount_amount + sfa.van_fee + sfa.previous_year_dues) AS total_due,
			COALESCE(SUM(fp.amount) FILTER (WHERE fp.voided = FALSE), 0) AS total_paid,
			(sfa.tuition_fee - sfa.discount_amount + sfa.van_fee + sfa.previous_year_dues)
				- COALESCE(SUM(fp.amount) FILTER (WHERE fp.voided = FALSE), 0) AS balance_remaining
		%s %s
		GROUP BY sfa.id, s.first_name, s.last_name, s.student_code, gl.name
		%s
		ORDER BY s.last_name, s.first_name
		LIMIT $%d OFFSET $%d`, countBase, where, having, argN, argN+1)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var items []FeeAccountSummary
	for rows.Next() {
		var s FeeAccountSummary
		if err := rows.Scan(&s.ID, &s.StudentID, &s.StudentName, &s.StudentCode, &s.GradeLevelName,
			&s.TuitionFee, &s.DiscountAmount, &s.VanFee, &s.PreviousYearDues, &s.IsRTE,
			&s.TotalDue, &s.TotalPaid, &s.BalanceRemaining); err != nil {
			return nil, 0, err
		}
		items = append(items, s)
	}
	return items, total, rows.Err()
}

func (r *Repository) UpdateFeeAccount(ctx context.Context, fa *domain.StudentFeeAccount) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE student_fee_accounts SET tuition_fee=$2, discount_amount=$3, discount_reason=$4,
			previous_year_dues=$5, van_fee=$6, is_rte=$7, updated_at=NOW()
		WHERE id=$1`,
		fa.ID, fa.TuitionFee, fa.DiscountAmount, fa.DiscountReason, fa.PreviousYearDues, fa.VanFee, fa.IsRTE,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return apperr.ErrNotFound
	}
	return nil
}

// ---- Payments ----

func (r *Repository) CreatePayment(ctx context.Context, p *domain.FeePayment) error {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO fee_payments (student_fee_account_id, fee_type, installment_number, amount, payment_date, payment_mode, reference_number, notes)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8) RETURNING id, created_at, updated_at`,
		p.StudentFeeAccountID, p.FeeType, p.InstallmentNumber, p.Amount, p.PaymentDate, p.PaymentMode, p.ReferenceNumber, p.Notes,
	)
	if err := row.Scan(&p.ID, &p.CreatedAt, &p.UpdatedAt); err != nil {
		return database.MapError(err)
	}
	return nil
}

func (r *Repository) ListPayments(ctx context.Context, accountID uuid.UUID) ([]domain.FeePayment, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, student_fee_account_id, fee_type, installment_number, amount, payment_date,
			payment_mode, COALESCE(reference_number,''), COALESCE(notes,''), voided, created_at, updated_at
		FROM fee_payments WHERE student_fee_account_id=$1 ORDER BY payment_date, created_at`, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []domain.FeePayment
	for rows.Next() {
		var p domain.FeePayment
		if err := rows.Scan(&p.ID, &p.StudentFeeAccountID, &p.FeeType, &p.InstallmentNumber, &p.Amount,
			&p.PaymentDate, &p.PaymentMode, &p.ReferenceNumber, &p.Notes, &p.Voided, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, p)
	}
	return items, rows.Err()
}

func (r *Repository) GetPaymentByID(ctx context.Context, id uuid.UUID) (*domain.FeePayment, error) {
	var p domain.FeePayment
	err := r.pool.QueryRow(ctx, `
		SELECT id, student_fee_account_id, fee_type, installment_number, amount, payment_date,
			payment_mode, COALESCE(reference_number,''), COALESCE(notes,''), voided, created_at, updated_at
		FROM fee_payments WHERE id=$1`, id,
	).Scan(&p.ID, &p.StudentFeeAccountID, &p.FeeType, &p.InstallmentNumber, &p.Amount,
		&p.PaymentDate, &p.PaymentMode, &p.ReferenceNumber, &p.Notes, &p.Voided, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperr.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get payment: %w", err)
	}
	return &p, nil
}

func (r *Repository) VoidPayment(ctx context.Context, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `UPDATE fee_payments SET voided=TRUE, updated_at=NOW() WHERE id=$1 AND voided=FALSE`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return apperr.ErrNotFound
	}
	return nil
}

// ---- Summaries ----

func (r *Repository) SchoolFeeSummary(ctx context.Context, schoolID, yearID uuid.UUID) (*SchoolFeeSummaryResponse, error) {
	resp := &SchoolFeeSummaryResponse{SchoolID: schoolID, AcademicYearID: yearID}

	err := r.pool.QueryRow(ctx, `
		SELECT COUNT(DISTINCT sfa.id),
			COUNT(DISTINCT sfa.id) FILTER (WHERE sfa.is_rte = TRUE),
			COALESCE(SUM(sfa.tuition_fee - sfa.discount_amount), 0),
			COALESCE(SUM(sfa.van_fee), 0),
			COALESCE(SUM(sfa.previous_year_dues), 0),
			COALESCE(SUM(sfa.discount_amount), 0),
			COALESCE(SUM(sfa.tuition_fee - sfa.discount_amount + sfa.van_fee + sfa.previous_year_dues), 0)
		FROM student_fee_accounts sfa
		WHERE sfa.school_id=$1 AND sfa.academic_year_id=$2`, schoolID, yearID,
	).Scan(&resp.TotalStudents, &resp.RTEStudents, &resp.TotalTuitionDue, &resp.TotalVanDue,
		&resp.TotalPrevDue, &resp.TotalDiscount, &resp.GrandTotalDue)
	if err != nil {
		return nil, err
	}

	err = r.pool.QueryRow(ctx, `
		SELECT COALESCE(SUM(fp.amount), 0)
		FROM fee_payments fp
		JOIN student_fee_accounts sfa ON sfa.id = fp.student_fee_account_id
		WHERE sfa.school_id=$1 AND sfa.academic_year_id=$2 AND fp.voided=FALSE`, schoolID, yearID,
	).Scan(&resp.TotalCollected)
	if err != nil {
		return nil, err
	}
	resp.TotalOutstanding = resp.GrandTotalDue - resp.TotalCollected

	rows, err := r.pool.Query(ctx, `
		SELECT gl.id, gl.name, COUNT(DISTINCT sfa.id),
			COALESCE(SUM(sfa.tuition_fee - sfa.discount_amount + sfa.van_fee + sfa.previous_year_dues), 0),
			COALESCE(SUM(paid.total), 0)
		FROM student_fee_accounts sfa
		JOIN fee_structures fs ON fs.id = sfa.fee_structure_id
		JOIN grade_levels gl ON gl.id = fs.grade_level_id
		LEFT JOIN LATERAL (
			SELECT COALESCE(SUM(fp.amount), 0) AS total
			FROM fee_payments fp
			WHERE fp.student_fee_account_id = sfa.id AND fp.voided = FALSE
		) paid ON TRUE
		WHERE sfa.school_id=$1 AND sfa.academic_year_id=$2
		GROUP BY gl.id, gl.name, gl.sort_order ORDER BY gl.sort_order`, schoolID, yearID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var g GradeFeeSummary
		if err := rows.Scan(&g.GradeLevelID, &g.GradeLevelName, &g.StudentCount, &g.TotalDue, &g.TotalCollected); err != nil {
			return nil, err
		}
		g.Outstanding = g.TotalDue - g.TotalCollected
		resp.ByGrade = append(resp.ByGrade, g)
	}
	return resp, rows.Err()
}

func (r *Repository) GetFeeAccountByStudent(ctx context.Context, studentID, yearID uuid.UUID) (*domain.StudentFeeAccount, error) {
	var fa domain.StudentFeeAccount
	err := r.pool.QueryRow(ctx, `
		SELECT id, student_id, school_id, academic_year_id, fee_structure_id,
			tuition_fee, discount_amount, COALESCE(discount_reason,''), previous_year_dues, van_fee, is_rte,
			created_at, updated_at
		FROM student_fee_accounts WHERE student_id=$1 AND academic_year_id=$2`, studentID, yearID,
	).Scan(&fa.ID, &fa.StudentID, &fa.SchoolID, &fa.AcademicYearID, &fa.FeeStructureID,
		&fa.TuitionFee, &fa.DiscountAmount, &fa.DiscountReason, &fa.PreviousYearDues, &fa.VanFee, &fa.IsRTE,
		&fa.CreatedAt, &fa.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperr.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get fee account by student: %w", err)
	}
	return &fa, nil
}

func (r *Repository) GetReceiptData(ctx context.Context, paymentID uuid.UUID) (*ReceiptData, error) {
	var d ReceiptData
	err := r.pool.QueryRow(ctx, `
		SELECT fp.id, fp.fee_type, fp.installment_number, fp.amount, fp.payment_date,
			fp.payment_mode, COALESCE(fp.reference_number,''), COALESCE(fp.notes,''),
			sch.name, COALESCE(sch.address,''), COALESCE(sch.phone,''), COALESCE(sch.email,''),
			s.first_name || ' ' || s.last_name, s.student_code,
			COALESCE(gl.name, ''),
			COALESCE(g.first_name || ' ' || g.last_name, ''),
			sfa.tuition_fee, sfa.discount_amount, sfa.van_fee, sfa.previous_year_dues,
			ay.name,
			sfa.id
		FROM fee_payments fp
		JOIN student_fee_accounts sfa ON sfa.id = fp.student_fee_account_id
		JOIN students s ON s.id = sfa.student_id
		JOIN schools sch ON sch.id = sfa.school_id
		JOIN fee_structures fs ON fs.id = sfa.fee_structure_id
		JOIN grade_levels gl ON gl.id = fs.grade_level_id
		JOIN academic_years ay ON ay.id = sfa.academic_year_id
		LEFT JOIN student_guardians sg ON sg.student_id = s.id AND sg.is_primary = TRUE
		LEFT JOIN guardians g ON g.id = sg.guardian_id
		WHERE fp.id = $1 AND fp.voided = FALSE`, paymentID,
	).Scan(&d.PaymentID, &d.FeeType, &d.InstallmentNum, &d.Amount, &d.PaymentDate,
		&d.PaymentMode, &d.ReferenceNumber, &d.Notes,
		&d.SchoolName, &d.SchoolAddress, &d.SchoolPhone, &d.SchoolEmail,
		&d.StudentName, &d.StudentCode,
		&d.GradeLevelName, &d.FatherName,
		&d.TuitionFee, &d.DiscountAmount, &d.VanFee, &d.PreviousYearDues,
		&d.AcademicYearName, &d.FeeAccountID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperr.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get receipt data: %w", err)
	}

	d.TotalDue = d.TuitionFee - d.DiscountAmount + d.VanFee + d.PreviousYearDues

	// Total paid before this payment
	err = r.pool.QueryRow(ctx, `
		SELECT COALESCE(SUM(amount), 0)
		FROM fee_payments
		WHERE student_fee_account_id = $1 AND voided = FALSE AND id != $2`, d.FeeAccountID, paymentID,
	).Scan(&d.TotalPaidOther)
	if err != nil {
		return nil, err
	}
	d.TotalPaidAfter = d.TotalPaidOther + d.Amount
	d.BalanceAfter = d.TotalDue - d.TotalPaidAfter

	return &d, nil
}

func (r *Repository) GetStudentInfo(ctx context.Context, studentID uuid.UUID) (name, code, gradeName string, err error) {
	err = r.pool.QueryRow(ctx, `
		SELECT s.first_name || ' ' || s.last_name, s.student_code, COALESCE(gl.name, '')
		FROM students s
		LEFT JOIN enrollments e ON e.student_id = s.id
		LEFT JOIN fee_structures fs ON fs.school_id = s.school_id AND fs.academic_year_id = e.academic_year_id
		LEFT JOIN grade_levels gl ON gl.id = fs.grade_level_id
		WHERE s.id = $1
		LIMIT 1`, studentID,
	).Scan(&name, &code, &gradeName)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", "", apperr.ErrNotFound
	}
	return
}
