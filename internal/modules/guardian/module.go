package guardian

import (
	"github.com/ajaypatel01/CampusDesk/internal/platform/httpx"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v4/pgxpool"
)

type Module struct {
	handler *Handler
}

func New(pool *pgxpool.Pool) *Module {
	repo := NewRepository(pool)
	svc := NewService(repo)
	return &Module{handler: NewHandler(svc)}
}

func (m *Module) Name() string { return "guardian" }

const feature = "guardians"

func (m *Module) Mount(r chi.Router) {
	h := m.handler
	view := httpx.RequireFeature(feature, "view")
	write := httpx.RequireFeature(feature, "write")
	r.Route("/guardians", func(r chi.Router) {
		r.With(view).Get("/", h.ListByStudent)
		r.With(write).Post("/", h.Create)
		r.With(write).Post("/link", h.Link)
		r.Route("/{id}", func(r chi.Router) {
			r.With(view).Get("/", h.Get)
		})
	})
}
