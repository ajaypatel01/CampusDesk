package whatsappbot

import (
	"github.com/ajaypatel01/CampusDesk/internal/modules/academic"
	"github.com/ajaypatel01/CampusDesk/internal/modules/fee"
	"github.com/ajaypatel01/CampusDesk/internal/modules/guardian"
	"github.com/ajaypatel01/CampusDesk/internal/modules/results"
	"github.com/ajaypatel01/CampusDesk/internal/modules/student"
	"github.com/ajaypatel01/CampusDesk/internal/platform/whatsapp"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v4/pgxpool"
)

type Module struct {
	handler *Handler
}

func New(pool *pgxpool.Pool, wa *whatsapp.Client) *Module {
	svc := NewService(
		guardian.NewRepository(pool), student.NewRepository(pool), academic.NewRepository(pool),
		fee.NewService(fee.NewRepository(pool)), results.NewRepository(pool), wa,
	)
	return &Module{handler: NewHandler(svc, wa)}
}

func (m *Module) Name() string { return "whatsappbot" }

// MountPublic mounts both webhook routes -- Meta calls these directly, never
// through a logged-in browser session, so neither can sit behind the
// JWT-protected route group. Both authenticate themselves independently
// (the verify-token handshake, then per-call signature verification).
func (m *Module) MountPublic(r chi.Router) {
	r.Get("/whatsapp/webhook", m.handler.VerifyWebhook)
	r.Post("/whatsapp/webhook", m.handler.ReceiveMessage)
}
