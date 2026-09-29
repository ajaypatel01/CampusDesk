package billing

import (
	"context"
	"encoding/json"
	"errors"
	"time"

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

func (r *Repository) CreatePlan(ctx context.Context, p *domain.SubscriptionPlan) error {
	features, err := json.Marshal(p.Features)
	if err != nil {
		return err
	}
	row := r.pool.QueryRow(ctx, `
		INSERT INTO subscription_plans
			(name, description, price_monthly_paise, price_annual_paise, max_students, features,
			 razorpay_plan_id_monthly, razorpay_plan_id_annual, is_active, sort_order)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		RETURNING id, created_at, updated_at`,
		p.Name, p.Description, p.PriceMonthlyPaise, p.PriceAnnualPaise, p.MaxStudents, features,
		p.RazorpayPlanIDMonthly, p.RazorpayPlanIDAnnual, p.IsActive, p.SortOrder,
	)
	return row.Scan(&p.ID, &p.CreatedAt, &p.UpdatedAt)
}

func (r *Repository) UpdatePlan(ctx context.Context, p *domain.SubscriptionPlan) error {
	features, err := json.Marshal(p.Features)
	if err != nil {
		return err
	}
	tag, err := r.pool.Exec(ctx, `
		UPDATE subscription_plans SET
			name=$2, description=$3, price_monthly_paise=$4, price_annual_paise=$5, max_students=$6,
			features=$7, razorpay_plan_id_monthly=$8, razorpay_plan_id_annual=$9, is_active=$10, sort_order=$11,
			updated_at=NOW()
		WHERE id=$1`,
		p.ID, p.Name, p.Description, p.PriceMonthlyPaise, p.PriceAnnualPaise, p.MaxStudents,
		features, p.RazorpayPlanIDMonthly, p.RazorpayPlanIDAnnual, p.IsActive, p.SortOrder,
	)
	if err != nil {
		return database.MapError(err)
	}
	if tag.RowsAffected() == 0 {
		return apperr.ErrNotFound
	}
	return nil
}

func scanPlan(row pgx.Row) (*domain.SubscriptionPlan, error) {
	var p domain.SubscriptionPlan
	var features []byte
	err := row.Scan(&p.ID, &p.Name, &p.Description, &p.PriceMonthlyPaise, &p.PriceAnnualPaise, &p.MaxStudents,
		&features, &p.RazorpayPlanIDMonthly, &p.RazorpayPlanIDAnnual, &p.IsActive, &p.SortOrder,
		&p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(features, &p.Features); err != nil {
		p.Features = nil
	}
	return &p, nil
}

const planColumns = `id, name, description, price_monthly_paise, price_annual_paise, max_students,
	features, razorpay_plan_id_monthly, razorpay_plan_id_annual, is_active, sort_order, created_at, updated_at`

func (r *Repository) ListPlans(ctx context.Context, activeOnly bool) ([]domain.SubscriptionPlan, error) {
	q := `SELECT ` + planColumns + ` FROM subscription_plans`
	if activeOnly {
		q += ` WHERE is_active = true`
	}
	q += ` ORDER BY sort_order, price_monthly_paise`
	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []domain.SubscriptionPlan
	for rows.Next() {
		p, err := scanPlan(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *p)
	}
	return items, rows.Err()
}

func (r *Repository) GetPlan(ctx context.Context, id uuid.UUID) (*domain.SubscriptionPlan, error) {
	p, err := scanPlan(r.pool.QueryRow(ctx, `SELECT `+planColumns+` FROM subscription_plans WHERE id=$1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperr.ErrNotFound
	}
	return p, err
}

const subscriptionColumns = `id, school_id, plan_id, billing_cycle, status, razorpay_subscription_id, razorpay_customer_id,
	current_period_start, current_period_end, cancel_at_cycle_end, cancelled_at, created_at, updated_at`

func scanSubscription(row pgx.Row) (*domain.Subscription, error) {
	var s domain.Subscription
	err := row.Scan(&s.ID, &s.SchoolID, &s.PlanID, &s.BillingCycle, &s.Status, &s.RazorpaySubscriptionID, &s.RazorpayCustomerID,
		&s.CurrentPeriodStart, &s.CurrentPeriodEnd, &s.CancelAtCycleEnd, &s.CancelledAt, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *Repository) GetSubscriptionBySchool(ctx context.Context, schoolID uuid.UUID) (*domain.Subscription, error) {
	s, err := scanSubscription(r.pool.QueryRow(ctx, `SELECT `+subscriptionColumns+` FROM subscriptions WHERE school_id=$1`, schoolID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperr.ErrNotFound
	}
	return s, err
}

func (r *Repository) GetSubscriptionByRazorpayID(ctx context.Context, razorpaySubscriptionID string) (*domain.Subscription, error) {
	s, err := scanSubscription(r.pool.QueryRow(ctx, `SELECT `+subscriptionColumns+` FROM subscriptions WHERE razorpay_subscription_id=$1`, razorpaySubscriptionID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperr.ErrNotFound
	}
	return s, err
}

// UpsertSubscription creates the school's subscription row, or replaces it
// in place if one already exists -- a school has at most one at a time (see
// the DB's unique index on school_id), so switching plans updates this row.
func (r *Repository) UpsertSubscription(ctx context.Context, s *domain.Subscription) error {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO subscriptions
			(school_id, plan_id, billing_cycle, status, razorpay_subscription_id, razorpay_customer_id,
			 current_period_start, current_period_end, cancel_at_cycle_end, cancelled_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		ON CONFLICT (school_id) DO UPDATE SET
			plan_id=$2, billing_cycle=$3, status=$4, razorpay_subscription_id=$5, razorpay_customer_id=$6,
			current_period_start=$7, current_period_end=$8, cancel_at_cycle_end=$9, cancelled_at=$10, updated_at=NOW()
		RETURNING id, created_at, updated_at`,
		s.SchoolID, s.PlanID, s.BillingCycle, s.Status, s.RazorpaySubscriptionID, s.RazorpayCustomerID,
		s.CurrentPeriodStart, s.CurrentPeriodEnd, s.CancelAtCycleEnd, s.CancelledAt,
	)
	return row.Scan(&s.ID, &s.CreatedAt, &s.UpdatedAt)
}

// UpdateSubscriptionStatus is the narrow update the webhook handler uses --
// it only ever learns a status (and sometimes a period), never a plan/cycle
// change, so it doesn't risk clobbering those fields with stale data.
func (r *Repository) UpdateSubscriptionStatus(ctx context.Context, razorpaySubscriptionID string, status domain.SubscriptionStatus, periodStart, periodEnd *time.Time) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE subscriptions SET status=$2,
			current_period_start = COALESCE($3, current_period_start),
			current_period_end = COALESCE($4, current_period_end),
			updated_at=NOW()
		WHERE razorpay_subscription_id=$1`,
		razorpaySubscriptionID, status, periodStart, periodEnd,
	)
	return err
}

func (r *Repository) MarkCancelled(ctx context.Context, id uuid.UUID, cancelAtCycleEnd bool) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE subscriptions SET cancel_at_cycle_end=$2, cancelled_at=NOW(), updated_at=NOW()
		WHERE id=$1`, id, cancelAtCycleEnd,
	)
	return err
}

func (r *Repository) CreatePayment(ctx context.Context, p *domain.SubscriptionPayment) error {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO subscription_payments (subscription_id, razorpay_payment_id, razorpay_invoice_id, amount_paise, status, paid_at)
		VALUES ($1,$2,$3,$4,$5,$6)
		ON CONFLICT (razorpay_payment_id) DO NOTHING
		RETURNING id, created_at`,
		p.SubscriptionID, p.RazorpayPaymentID, p.RazorpayInvoiceID, p.AmountPaise, p.Status, p.PaidAt,
	)
	return row.Scan(&p.ID, &p.CreatedAt)
}

func (r *Repository) ListPaymentsBySubscription(ctx context.Context, subscriptionID uuid.UUID) ([]domain.SubscriptionPayment, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, subscription_id, razorpay_payment_id, razorpay_invoice_id, amount_paise, status, paid_at, created_at
		FROM subscription_payments WHERE subscription_id=$1 ORDER BY created_at DESC`, subscriptionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []domain.SubscriptionPayment
	for rows.Next() {
		var p domain.SubscriptionPayment
		if err := rows.Scan(&p.ID, &p.SubscriptionID, &p.RazorpayPaymentID, &p.RazorpayInvoiceID, &p.AmountPaise, &p.Status, &p.PaidAt, &p.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, p)
	}
	return items, rows.Err()
}
