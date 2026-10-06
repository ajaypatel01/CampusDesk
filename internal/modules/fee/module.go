package fee

import (
	"github.com/ajaypatel01/CampusDesk/internal/modules/guardian"
	"github.com/ajaypatel01/CampusDesk/internal/platform/httpx"
	"github.com/ajaypatel01/CampusDesk/internal/platform/whatsapp"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v4/pgxpool"
)

type Module struct {
	handler *Handler
}

func New(pool *pgxpool.Pool, wa *whatsapp.Client) *Module {
	repo := NewRepository(pool)
	svc := NewService(repo)
	return &Module{handler: NewHandler(svc, wa, guardian.NewRepository(pool))}
}

func (m *Module) Name() string { return "fee" }

const feature = "fees"

func (m *Module) Mount(r chi.Router) {
	h := m.handler
	view := httpx.RequireFeature(feature, "view")
	write := httpx.RequireFeature(feature, "write")

	// Editing fee amounts is limited to admins and registrars; everyone else stays read-only.
	canEditFees := httpx.RequireRole("super_admin", "school_admin", "registrar")

	// Fee data is financial/admin information, not something a class
	// teacher needs to enter results or assign homework.
	r.Group(func(r chi.Router) {
		r.Use(httpx.BlockRoles("teacher"))

		r.Route("/fee-structures", func(r chi.Router) {
			r.With(view).Get("/", h.ListFeeStructures)
			r.With(canEditFees, write).Post("/", h.CreateFeeStructure)
			r.Route("/{id}", func(r chi.Router) {
				r.With(view).Get("/", h.GetFeeStructure)
				r.With(canEditFees, write).Put("/", h.UpdateFeeStructure)
			})
		})

		r.Route("/fee-accounts", func(r chi.Router) {
			r.With(view).Get("/", h.ListFeeAccounts)
			r.With(canEditFees, write).Post("/", h.CreateFeeAccount)
			r.Route("/{id}", func(r chi.Router) {
				r.With(view).Get("/", h.GetFeeAccount)
				r.With(canEditFees, write).Put("/", h.UpdateFeeAccount)
			})
		})

		r.Route("/fee-payments", func(r chi.Router) {
			r.With(view).Get("/", h.ListPayments)
			// The whole school's payments with student names: office staff only.
			r.With(httpx.BlockRoles("teacher", "parent"), view).Get("/ledger", h.ListLedgerPayments)
			r.With(write).Post("/", h.RecordPayment)
			r.With(write).Delete("/{id}", h.VoidPayment)
			// Moving a payment to a different year corrects a mis-entered record
			// (e.g. a 2025-26 payment logged under 2026-27) rather than editing
			// its amount/date, so it's scoped more tightly than canEditFees --
			// registrar and super_admin only, not school_admin.
			r.With(httpx.RequireRole("super_admin", "registrar"), write).Put("/{id}/move", h.MovePayment)
		})

		r.Route("/fee-receipts", func(r chi.Router) {
			r.With(view).Get("/{payment_id}", h.DownloadReceipt)
			r.With(write).Post("/{payment_id}/whatsapp", h.SendReceiptWhatsApp)
		})

		r.Route("/fee-summary", func(r chi.Router) {
			// School-wide aggregate totals are hidden from registrars; per-student summaries stay visible.
			r.With(httpx.BlockRoles("registrar"), view).Get("/", h.SchoolFeeSummary)
			r.With(view).Get("/student/{student_id}", h.StudentFeeSummary)
		})

		// Installment sheet is a school-wide aggregate view, hidden from registrars like fee-summary.
		r.With(httpx.BlockRoles("registrar"), view).Get("/fee-installment-sheet", h.InstallmentSheet)
	})
}
