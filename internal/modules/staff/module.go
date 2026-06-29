package staff

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

func (m *Module) Name() string { return "staff" }

func (m *Module) Mount(r chi.Router) {
	r.Route("/staff", func(r chi.Router) {
		r.Get("/", m.handler.List)
		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", m.handler.Get)
			r.Put("/profile", m.handler.UpsertProfile)
		})
	})
}
