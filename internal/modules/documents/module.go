package documents

import (
	"github.com/ajaypatel01/CampusDesk/internal/platform/email"
	"github.com/ajaypatel01/CampusDesk/internal/platform/httpx"
	"github.com/ajaypatel01/CampusDesk/internal/platform/whatsapp"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v4/pgxpool"
)

type Module struct {
	handler *Handler
}

func New(pool *pgxpool.Pool, emailClient *email.Client, waClient *whatsapp.Client) *Module {
	repo := NewRepository(pool)
	svc := NewService(repo)
	return &Module{handler: NewHandler(svc, emailClient, waClient)}
}

func (m *Module) Name() string { return "documents" }

const feature = "documents"

func (m *Module) Mount(r chi.Router) {
	h := m.handler
	view := httpx.RequireFeature(feature, "view")
	write := httpx.RequireFeature(feature, "write")
	r.Route("/documents", func(r chi.Router) {
		// Generating bonafide/TC/salary-slip documents for *other* people is
		// admin/registrar work. A teacher's own salary slip comes from the
		// payroll module instead (self-scoped there), not this admin path.
		r.Use(httpx.BlockRoles("registrar", "teacher"))
		r.With(view).Get("/bonafide", h.DownloadBonafide)
		r.With(write).Post("/bonafide/email", h.EmailBonafide)
		r.With(write).Post("/bonafide/whatsapp", h.WhatsAppBonafide)
		r.With(view).Get("/transfer-certificate", h.DownloadTC)
		r.With(write).Post("/transfer-certificate/email", h.EmailTC)
		r.With(write).Post("/transfer-certificate/whatsapp", h.WhatsAppTC)
		// DownloadSalarySlip is a POST (form-carrying) request but is
		// conceptually a read -- generating and returning a PDF, no state
		// change -- so it's gated as "view" like the other Download* routes.
		r.With(view).Post("/salary-slip", h.DownloadSalarySlip)
		r.With(write).Post("/salary-slip/email", h.EmailSalarySlip)
		r.With(write).Post("/salary-slip/whatsapp", h.WhatsAppSalarySlip)
	})
}
