// Package billing is CampusDesk's own SaaS subscription billing -- what a
// school pays CampusDesk to use the app -- via Razorpay Subscriptions.
// Deliberately separate from internal/modules/fee, which is a school
// collecting tuition fees from its own students/parents.
//
// This module does NOT gate access to anything: a school with a lapsed or
// cancelled subscription still fully works. Enforcement (if wanted later)
// is a separate, deliberate decision -- this just tracks and collects
// payment.
package billing

import (
	"github.com/ajaypatel01/CampusDesk/internal/platform/httpx"
	"github.com/ajaypatel01/CampusDesk/internal/platform/razorpay"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v4/pgxpool"
)

type Module struct {
	handler *Handler
}

func New(pool *pgxpool.Pool, rzp *razorpay.Client, razorpayKeyID string) *Module {
	repo := NewRepository(pool)
	svc := NewService(repo, rzp)
	return &Module{handler: NewHandler(svc, razorpayKeyID)}
}

func (m *Module) Name() string { return "billing" }

// MountPublic mounts the webhook -- Razorpay calls this server-to-server,
// never through a logged-in browser session, so it can't sit behind the
// JWT-protected route group. It authenticates itself via signature instead.
func (m *Module) MountPublic(r chi.Router) {
	r.Post("/billing/webhook", m.handler.Webhook)
}

// Mount is everything a logged-in admin uses: viewing plans, subscribing,
// cancelling, and (super_admin only) managing what plans exist. Restricted
// to school_admin/super_admin -- teachers, registrars, and parents have no
// reason to see or touch a school's own CampusDesk subscription.
func (m *Module) Mount(r chi.Router) {
	h := m.handler
	adminOnly := httpx.BlockRoles("teacher", "registrar", "parent")
	superAdminOnly := httpx.RequireRole("super_admin")
	r.Route("/billing", func(r chi.Router) {
		r.With(adminOnly).Get("/plans", h.ListPlans)
		r.With(adminOnly).Get("/subscription", h.GetSubscription)
		r.With(adminOnly).Post("/subscribe", h.Subscribe)
		r.With(adminOnly).Post("/confirm", h.ConfirmCheckout)
		r.With(adminOnly).Post("/cancel", h.Cancel)
		// Defining what plans/prices exist at all is platform-level
		// configuration, not a per-school admin action.
		r.With(superAdminOnly).Get("/plans/all", h.ListAllPlans)
		r.With(superAdminOnly).Post("/plans", h.CreatePlan)
		r.With(superAdminOnly).Put("/plans/{id}", h.UpdatePlan)
	})
}
