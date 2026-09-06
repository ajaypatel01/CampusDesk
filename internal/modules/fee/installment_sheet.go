package fee

import (
	"context"
	"net/http"

	"github.com/ajaypatel01/CampusDesk/internal/platform/httpx"
	"github.com/google/uuid"
)

// ---- Types ----

type InstallmentCell struct {
	InstallmentNumber int    `json:"installment_number"`
	Label             string `json:"label"`
	DueDate           string `json:"due_date,omitempty"`
	PlannedAmount     int    `json:"planned_amount"`
	PaidAmount        int    `json:"paid_amount"`
	Status            string `json:"status"` // paid | partial | pending
}

type StudentInstallmentRow struct {
	StudentID      uuid.UUID          `json:"student_id"`
	StudentName    string             `json:"student_name"`
	StudentCode    string             `json:"student_code"`
	GradeLevelName string             `json:"grade_level_name"`
	TotalDue       int                `json:"total_due"`
	TotalPaid      int                `json:"total_paid"`
	Balance        int                `json:"balance"`
	Installments   []InstallmentCell  `json:"installments"`
}

type InstallmentSheetResponse struct {
	Items                 []StudentInstallmentRow `json:"items"`
	MaxInstallments       int                     `json:"max_installments"`
	TotalStudents         int                     `json:"total_students"`
	Installment1PaidCount int                     `json:"installment1_paid_count"`
}

// ---- Repository ----

type installmentAccountRow struct {
	studentID        uuid.UUID
	firstName        string
	lastName         string
	studentCode      string
	gradeLevelName   string
	accountID        uuid.UUID
	tuitionFee       int
	discountAmount   int
	vanFee           int
	previousYearDues int
	feeStructureID   uuid.UUID
	numInstallments  int
}

func (r *Repository) listInstallmentAccounts(ctx context.Context, schoolID, yearID uuid.UUID, gradeLevelID *uuid.UUID) ([]installmentAccountRow, error) {
	q := `
		SELECT s.id, s.first_name, s.last_name, s.student_code, gl.name,
			sfa.id, sfa.tuition_fee, sfa.discount_amount, sfa.van_fee, sfa.previous_year_dues,
			fs.id, fs.num_installments
		FROM student_fee_accounts sfa
		JOIN students s ON s.id = sfa.student_id
		JOIN fee_structures fs ON fs.id = sfa.fee_structure_id
		JOIN grade_levels gl ON gl.id = fs.grade_level_id
		WHERE sfa.school_id=$1 AND sfa.academic_year_id=$2`
	args := []interface{}{schoolID, yearID}
	if gradeLevelID != nil {
		q += " AND fs.grade_level_id=$3"
		args = append(args, *gradeLevelID)
	}
	q += " ORDER BY gl.sort_order, s.first_name, s.last_name"

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []installmentAccountRow
	for rows.Next() {
		var a installmentAccountRow
		if err := rows.Scan(&a.studentID, &a.firstName, &a.lastName, &a.studentCode, &a.gradeLevelName,
			&a.accountID, &a.tuitionFee, &a.discountAmount, &a.vanFee, &a.previousYearDues,
			&a.feeStructureID, &a.numInstallments); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

type installmentPlanRow struct {
	feeStructureID    uuid.UUID
	installmentNumber int
	label             string
	amount            int
	dueDate           string
}

func (r *Repository) listInstallmentPlansFor(ctx context.Context, feeStructureIDs []uuid.UUID) ([]installmentPlanRow, error) {
	if len(feeStructureIDs) == 0 {
		return nil, nil
	}
	rows, err := r.pool.Query(ctx, `
		SELECT fee_structure_id, installment_number, label, amount, to_char(due_date, 'YYYY-MM-DD')
		FROM fee_installment_plans WHERE fee_structure_id = ANY($1) ORDER BY installment_number`,
		feeStructureIDs,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []installmentPlanRow
	for rows.Next() {
		var p installmentPlanRow
		if err := rows.Scan(&p.feeStructureID, &p.installmentNumber, &p.label, &p.amount, &p.dueDate); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

type installmentPaidRow struct {
	accountID         uuid.UUID
	installmentNumber int
	paid              int
}

// sumTotalPaidByAccount returns each account's total non-voided payments across every
// fee type (tuition + van etc.), used for the Total Paid/Balance columns — the
// installment-cell breakdown below is tuition-only since only tuition ties to a plan.
func (r *Repository) sumTotalPaidByAccount(ctx context.Context, accountIDs []uuid.UUID) (map[uuid.UUID]int, error) {
	if len(accountIDs) == 0 {
		return map[uuid.UUID]int{}, nil
	}
	rows, err := r.pool.Query(ctx, `
		SELECT student_fee_account_id, SUM(amount)
		FROM fee_payments
		WHERE student_fee_account_id = ANY($1) AND voided=FALSE
		GROUP BY student_fee_account_id`,
		accountIDs,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[uuid.UUID]int{}
	for rows.Next() {
		var id uuid.UUID
		var total int
		if err := rows.Scan(&id, &total); err != nil {
			return nil, err
		}
		out[id] = total
	}
	return out, rows.Err()
}

func (r *Repository) sumTuitionPaidByInstallment(ctx context.Context, accountIDs []uuid.UUID) ([]installmentPaidRow, error) {
	if len(accountIDs) == 0 {
		return nil, nil
	}
	rows, err := r.pool.Query(ctx, `
		SELECT student_fee_account_id, installment_number, SUM(amount)
		FROM fee_payments
		WHERE student_fee_account_id = ANY($1) AND fee_type='tuition' AND voided=FALSE AND installment_number IS NOT NULL
		GROUP BY student_fee_account_id, installment_number`,
		accountIDs,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []installmentPaidRow
	for rows.Next() {
		var p installmentPaidRow
		if err := rows.Scan(&p.accountID, &p.installmentNumber, &p.paid); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// ---- Service ----

func (s *Service) InstallmentSheet(ctx context.Context, schoolID, yearID uuid.UUID, gradeLevelID *uuid.UUID) (*InstallmentSheetResponse, error) {
	accounts, err := s.repo.listInstallmentAccounts(ctx, schoolID, yearID, gradeLevelID)
	if err != nil {
		return nil, err
	}

	feeStructureSet := map[uuid.UUID]bool{}
	accountIDs := make([]uuid.UUID, 0, len(accounts))
	for _, a := range accounts {
		feeStructureSet[a.feeStructureID] = true
		accountIDs = append(accountIDs, a.accountID)
	}
	feeStructureIDs := make([]uuid.UUID, 0, len(feeStructureSet))
	for id := range feeStructureSet {
		feeStructureIDs = append(feeStructureIDs, id)
	}

	plans, err := s.repo.listInstallmentPlansFor(ctx, feeStructureIDs)
	if err != nil {
		return nil, err
	}
	plansByStructure := map[uuid.UUID][]installmentPlanRow{}
	for _, p := range plans {
		plansByStructure[p.feeStructureID] = append(plansByStructure[p.feeStructureID], p)
	}

	paidRows, err := s.repo.sumTuitionPaidByInstallment(ctx, accountIDs)
	if err != nil {
		return nil, err
	}
	paidByAccount := map[uuid.UUID]map[int]int{}
	for _, p := range paidRows {
		if paidByAccount[p.accountID] == nil {
			paidByAccount[p.accountID] = map[int]int{}
		}
		paidByAccount[p.accountID][p.installmentNumber] = p.paid
	}

	totalPaidByAccount, err := s.repo.sumTotalPaidByAccount(ctx, accountIDs)
	if err != nil {
		return nil, err
	}

	resp := &InstallmentSheetResponse{}
	maxInstallments := 0
	for _, a := range accounts {
		totalDue := a.tuitionFee - a.discountAmount + a.vanFee + a.previousYearDues
		accountPaid := paidByAccount[a.accountID]

		cells := make([]InstallmentCell, 0, a.numInstallments)
		for n := 1; n <= a.numInstallments; n++ {
			var label string
			var amount int
			var dueDate string
			for _, p := range plansByStructure[a.feeStructureID] {
				if p.installmentNumber == n {
					label = p.label
					amount = p.amount
					dueDate = p.dueDate
					break
				}
			}
			if label == "" {
				label = "Installment"
			}
			paid := accountPaid[n]
			status := "pending"
			if amount > 0 && paid >= amount {
				status = "paid"
			} else if paid > 0 {
				status = "partial"
			}
			cells = append(cells, InstallmentCell{
				InstallmentNumber: n,
				Label:             label,
				DueDate:           dueDate,
				PlannedAmount:     amount,
				PaidAmount:        paid,
				Status:            status,
			})
			if paid > 0 && n == 1 {
				resp.Installment1PaidCount++
			}
		}
		if a.numInstallments > maxInstallments {
			maxInstallments = a.numInstallments
		}

		totalPaid := totalPaidByAccount[a.accountID]

		resp.Items = append(resp.Items, StudentInstallmentRow{
			StudentID:      a.studentID,
			StudentName:    a.firstName + " " + a.lastName,
			StudentCode:    a.studentCode,
			GradeLevelName: a.gradeLevelName,
			TotalDue:       totalDue,
			TotalPaid:      totalPaid,
			Balance:        totalDue - totalPaid,
			Installments:   cells,
		})
	}
	resp.MaxInstallments = maxInstallments
	resp.TotalStudents = len(accounts)
	return resp, nil
}

// ---- Handler ----

func (h *Handler) InstallmentSheet(w http.ResponseWriter, r *http.Request) {
	schoolID, err := uuid.Parse(r.URL.Query().Get("school_id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "school_id required")
		return
	}
	yearID, err := uuid.Parse(r.URL.Query().Get("academic_year_id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "academic_year_id required")
		return
	}
	var gradeLevelID *uuid.UUID
	if g := r.URL.Query().Get("grade_level_id"); g != "" {
		id, err := uuid.Parse(g)
		if err != nil {
			httpx.Error(w, http.StatusBadRequest, "invalid grade_level_id")
			return
		}
		gradeLevelID = &id
	}
	resp, err := h.svc.InstallmentSheet(r.Context(), schoolID, yearID, gradeLevelID)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}
