package billing

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ajaypatel01/CampusDesk/internal/domain"
	apperr "github.com/ajaypatel01/CampusDesk/internal/platform/errors"
	"github.com/ajaypatel01/CampusDesk/internal/platform/razorpay"
	"github.com/google/uuid"
)

type Service struct {
	repo *Repository
	rzp  *razorpay.Client
}

func NewService(repo *Repository, rzp *razorpay.Client) *Service {
	return &Service{repo: repo, rzp: rzp}
}

func (s *Service) Enabled() bool { return s.rzp.Enabled() }

type PlanInput struct {
	Name                  string   `json:"name"`
	Description           string   `json:"description"`
	PriceMonthlyPaise     int      `json:"price_monthly_paise"`
	PriceAnnualPaise      int      `json:"price_annual_paise"`
	MaxStudents           *int     `json:"max_students"`
	Features              []string `json:"features"`
	RazorpayPlanIDMonthly string   `json:"razorpay_plan_id_monthly"`
	RazorpayPlanIDAnnual  string   `json:"razorpay_plan_id_annual"`
	IsActive              bool     `json:"is_active"`
	SortOrder             int      `json:"sort_order"`
}

func (s *Service) CreatePlan(ctx context.Context, in PlanInput) (*domain.SubscriptionPlan, error) {
	if strings.TrimSpace(in.Name) == "" {
		return nil, apperr.ErrInvalidInput
	}
	p := &domain.SubscriptionPlan{
		Name: strings.TrimSpace(in.Name), Description: in.Description,
		PriceMonthlyPaise: in.PriceMonthlyPaise, PriceAnnualPaise: in.PriceAnnualPaise,
		MaxStudents: in.MaxStudents, Features: in.Features,
		RazorpayPlanIDMonthly: in.RazorpayPlanIDMonthly, RazorpayPlanIDAnnual: in.RazorpayPlanIDAnnual,
		IsActive: in.IsActive, SortOrder: in.SortOrder,
	}
	if err := s.repo.CreatePlan(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *Service) UpdatePlan(ctx context.Context, id uuid.UUID, in PlanInput) (*domain.SubscriptionPlan, error) {
	if strings.TrimSpace(in.Name) == "" {
		return nil, apperr.ErrInvalidInput
	}
	p := &domain.SubscriptionPlan{
		ID: id, Name: strings.TrimSpace(in.Name), Description: in.Description,
		PriceMonthlyPaise: in.PriceMonthlyPaise, PriceAnnualPaise: in.PriceAnnualPaise,
		MaxStudents: in.MaxStudents, Features: in.Features,
		RazorpayPlanIDMonthly: in.RazorpayPlanIDMonthly, RazorpayPlanIDAnnual: in.RazorpayPlanIDAnnual,
		IsActive: in.IsActive, SortOrder: in.SortOrder,
	}
	if err := s.repo.UpdatePlan(ctx, p); err != nil {
		return nil, err
	}
	return s.repo.GetPlan(ctx, id)
}

func (s *Service) ListPlans(ctx context.Context, activeOnly bool) ([]domain.SubscriptionPlan, error) {
	return s.repo.ListPlans(ctx, activeOnly)
}

func (s *Service) GetSubscription(ctx context.Context, schoolID uuid.UUID) (*domain.Subscription, error) {
	return s.repo.GetSubscriptionBySchool(ctx, schoolID)
}

func (s *Service) ListPayments(ctx context.Context, subscriptionID uuid.UUID) ([]domain.SubscriptionPayment, error) {
	return s.repo.ListPaymentsBySubscription(ctx, subscriptionID)
}

// SubscribeResult is what the frontend needs to open Razorpay Checkout.
type SubscribeResult struct {
	Subscription           *domain.Subscription `json:"subscription"`
	RazorpaySubscriptionID string               `json:"razorpay_subscription_id"`
	RazorpayKeyID          string               `json:"razorpay_key_id"`
}

// Subscribe creates (or replaces) a school's subscription: a Razorpay
// subscription is created first against the plan's matching Razorpay plan
// ID for the chosen billing cycle, then mirrored locally with whatever
// status Razorpay returned (normally "created", pending the customer
// actually completing checkout). The webhook handler is what later flips
// this to "active" once payment succeeds.
func (s *Service) Subscribe(ctx context.Context, schoolID, planID uuid.UUID, billingCycle string) (*SubscribeResult, error) {
	if !s.rzp.Enabled() {
		return nil, fmt.Errorf("%w: Razorpay isn't configured yet -- add RAZORPAY_KEY_ID/RAZORPAY_KEY_SECRET first", apperr.ErrInvalidInput)
	}
	if billingCycle != "monthly" && billingCycle != "annual" {
		return nil, fmt.Errorf("%w: billing_cycle must be monthly or annual", apperr.ErrInvalidInput)
	}
	plan, err := s.repo.GetPlan(ctx, planID)
	if err != nil {
		return nil, err
	}
	razorpayPlanID := plan.RazorpayPlanIDMonthly
	totalCount := 12 // Razorpay requires a finite total_count; 12 monthly cycles = renew yearly by re-subscribing
	if billingCycle == "annual" {
		razorpayPlanID = plan.RazorpayPlanIDAnnual
		totalCount = 5 // 5 years, generous upper bound; cancel_at_cycle_end or re-subscribe covers changes
	}
	if razorpayPlanID == "" {
		return nil, fmt.Errorf("%w: this plan has no Razorpay %s plan ID set yet -- create one on the Razorpay dashboard first and add it under Settings", apperr.ErrInvalidInput, billingCycle)
	}

	rzpSub, err := s.rzp.CreateSubscription(razorpay.CreateSubscriptionInput{
		RazorpayPlanID: razorpayPlanID, TotalCount: totalCount, CustomerNotify: true,
		Notes: map[string]string{"school_id": schoolID.String(), "plan_id": planID.String()},
	})
	if err != nil {
		return nil, fmt.Errorf("razorpay: %w", err)
	}

	sub := &domain.Subscription{
		SchoolID: schoolID, PlanID: planID, BillingCycle: billingCycle,
		Status:                 domain.SubscriptionStatus(rzpSub.Status),
		RazorpaySubscriptionID: rzpSub.ID,
	}
	if err := s.repo.UpsertSubscription(ctx, sub); err != nil {
		return nil, err
	}
	return &SubscribeResult{Subscription: sub, RazorpaySubscriptionID: rzpSub.ID}, nil
}

// ConfirmCheckout verifies the signature Razorpay Checkout returned to the
// browser and, if valid, optimistically marks the subscription active --
// belt-and-braces alongside the webhook, which is the authoritative source
// of truth and will correct this if it's ever wrong.
func (s *Service) ConfirmCheckout(ctx context.Context, razorpaySubscriptionID, paymentID, signature string) error {
	if !s.rzp.VerifyCheckoutSignature(razorpaySubscriptionID, paymentID, signature) {
		return fmt.Errorf("%w: signature verification failed", apperr.ErrInvalidInput)
	}
	now := time.Now()
	return s.repo.UpdateSubscriptionStatus(ctx, razorpaySubscriptionID, domain.SubscriptionStatusActive, &now, nil)
}

func (s *Service) Cancel(ctx context.Context, schoolID uuid.UUID, atCycleEnd bool) error {
	sub, err := s.repo.GetSubscriptionBySchool(ctx, schoolID)
	if err != nil {
		return err
	}
	if sub.RazorpaySubscriptionID != "" && s.rzp.Enabled() {
		if err := s.rzp.CancelSubscription(sub.RazorpaySubscriptionID, atCycleEnd); err != nil {
			return fmt.Errorf("razorpay: %w", err)
		}
	}
	return s.repo.MarkCancelled(ctx, sub.ID, atCycleEnd)
}

// HandleWebhookEvent applies one Razorpay subscription/payment event. Events
// this doesn't recognize are ignored rather than erroring -- Razorpay sends
// many event types this app doesn't need to react to, and a 200 response is
// what stops Razorpay from retrying.
func (s *Service) HandleWebhookEvent(ctx context.Context, event WebhookEvent) error {
	switch event.Event {
	case "subscription.activated", "subscription.charged", "subscription.pending", "subscription.halted",
		"subscription.cancelled", "subscription.completed":
		sub := event.Payload.Subscription.Entity
		if sub.ID == "" {
			return nil
		}
		status := domain.SubscriptionStatus(sub.Status)
		var periodStart, periodEnd *time.Time
		if sub.CurrentStart > 0 {
			t := time.Unix(sub.CurrentStart, 0)
			periodStart = &t
		}
		if sub.CurrentEnd > 0 {
			t := time.Unix(sub.CurrentEnd, 0)
			periodEnd = &t
		}
		if err := s.repo.UpdateSubscriptionStatus(ctx, sub.ID, status, periodStart, periodEnd); err != nil {
			return err
		}
		// A charge event also carries the payment that triggered it.
		if event.Event == "subscription.charged" && event.Payload.Payment.Entity.ID != "" {
			return s.recordPayment(ctx, sub.ID, event.Payload.Payment.Entity)
		}
		return nil
	case "payment.failed":
		if event.Payload.Payment.Entity.ID == "" || event.Payload.Payment.Entity.SubscriptionID == "" {
			return nil
		}
		return s.recordPayment(ctx, event.Payload.Payment.Entity.SubscriptionID, event.Payload.Payment.Entity)
	default:
		return nil
	}
}

func (s *Service) recordPayment(ctx context.Context, razorpaySubscriptionID string, p WebhookPaymentEntity) error {
	sub, err := s.repo.GetSubscriptionByRazorpayID(ctx, razorpaySubscriptionID)
	if err != nil {
		return err
	}
	status := "captured"
	var paidAt *time.Time
	if p.Status == "failed" {
		status = "failed"
	} else if p.CreatedAt > 0 {
		t := time.Unix(p.CreatedAt, 0)
		paidAt = &t
	}
	return s.repo.CreatePayment(ctx, &domain.SubscriptionPayment{
		SubscriptionID: sub.ID, RazorpayPaymentID: p.ID, AmountPaise: p.Amount, Status: status, PaidAt: paidAt,
	})
}
