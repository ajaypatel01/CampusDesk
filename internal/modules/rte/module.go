package rte

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

func (m *Module) Name() string { return "rte" }

func (m *Module) Mount(r chi.Router) {
	h := m.handler

	r.Route("/rte", func(r chi.Router) {
		r.Get("/summary", h.GetSummary)
		r.Get("/students", h.ListRTEStudents)
		r.Route("/quotas", func(r chi.Router) {
			r.Get("/", h.ListQuotas)
			r.Post("/", h.UpsertQuota)
			r.Delete("/{id}", h.DeleteQuota)
		})
	})
}
