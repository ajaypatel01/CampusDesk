package domain

import (
	"time"

	"github.com/google/uuid"
)

// SubscriptionPlan is a billing tier CampusDesk offers to schools -- separate
// from the fee module's Subject/FeeStructure, which is about a school
// collecting fees from its own students, not what a school pays CampusDesk.
type SubscriptionPlan struct {
	ID                    uuid.UUID `json:"id"`
	Name                  string    `json:"name"`
	Description           string    `json:"description,omitempty"`
	PriceMonthlyPaise     int       `json:"price_monthly_paise"`
	PriceAnnualPaise      int       `json:"price_annual_paise"`
	MaxStudents           *int      `json:"max_students,omitempty"` // nil = unlimited
	Features              []string  `json:"features"`
	RazorpayPlanIDMonthly string    `json:"razorpay_plan_id_monthly,omitempty"`
	RazorpayPlanIDAnnual  string    `json:"razorpay_plan_id_annual,omitempty"`
	IsActive              bool      `json:"is_active"`
	SortOrder             int       `json:"sort_order"`
	Timestamps
}

type SubscriptionStatus string

const (
	SubscriptionStatusCreated       SubscriptionStatus = "created"
	SubscriptionStatusAuthenticated SubscriptionStatus = "authenticated"
	SubscriptionStatusActive        SubscriptionStatus = "active"
	SubscriptionStatusPending       SubscriptionStatus = "pending"
	SubscriptionStatusHalted        SubscriptionStatus = "halted"
	SubscriptionStatusCancelled     SubscriptionStatus = "cancelled"
	SubscriptionStatusCompleted     SubscriptionStatus = "completed"
	SubscriptionStatusExpired       SubscriptionStatus = "expired"
)

// Subscription is one school's current billing relationship with
// CampusDesk. There's at most one per school (see the DB's unique index) --
// changing plans updates this row rather than creating another.
type Subscription struct {
	ID                     uuid.UUID          `json:"id"`
	SchoolID               uuid.UUID          `json:"school_id"`
	PlanID                 uuid.UUID          `json:"plan_id"`
	BillingCycle           string             `json:"billing_cycle"` // "monthly" | "annual"
	Status                 SubscriptionStatus `json:"status"`
	RazorpaySubscriptionID string             `json:"razorpay_subscription_id,omitempty"`
	RazorpayCustomerID     string             `json:"razorpay_customer_id,omitempty"`
	CurrentPeriodStart     *time.Time         `json:"current_period_start,omitempty"`
	CurrentPeriodEnd       *time.Time         `json:"current_period_end,omitempty"`
	CancelAtCycleEnd       bool               `json:"cancel_at_cycle_end"`
	CancelledAt            *time.Time         `json:"cancelled_at,omitempty"`
	Timestamps
}

type SubscriptionPayment struct {
	ID                uuid.UUID  `json:"id"`
	SubscriptionID    uuid.UUID  `json:"subscription_id"`
	RazorpayPaymentID string     `json:"razorpay_payment_id,omitempty"`
	RazorpayInvoiceID string     `json:"razorpay_invoice_id,omitempty"`
	AmountPaise       int        `json:"amount_paise"`
	Status            string     `json:"status"` // "captured" | "failed" | "refunded"
	PaidAt            *time.Time `json:"paid_at,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
}
