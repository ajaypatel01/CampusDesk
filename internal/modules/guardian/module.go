package guardian

import (
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

func (m *Module) Mount(r chi.Router) {
	h := m.handler
	r.Route("/guardians", func(r chi.Router) {
		r.Get("/", h.ListByStudent)
		r.Post("/", h.Create)
		r.Post("/link", h.Link)
		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", h.Get)
		})
	})
}
