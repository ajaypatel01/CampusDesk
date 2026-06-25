package documents

import (
	"github.com/ajaypatel01/CampusDesk/internal/platform/email"
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

func (m *Module) Mount(r chi.Router) {
	h := m.handler
	r.Route("/documents", func(r chi.Router) {
		r.Get("/bonafide", h.DownloadBonafide)
		r.Post("/bonafide/email", h.EmailBonafide)
		r.Post("/bonafide/whatsapp", h.WhatsAppBonafide)
		r.Get("/transfer-certificate", h.DownloadTC)
		r.Post("/transfer-certificate/email", h.EmailTC)
		r.Post("/transfer-certificate/whatsapp", h.WhatsAppTC)
		r.Post("/salary-slip", h.DownloadSalarySlip)
		r.Post("/salary-slip/email", h.EmailSalarySlip)
		r.Post("/salary-slip/whatsapp", h.WhatsAppSalarySlip)
	})
}
