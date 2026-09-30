package rte

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

func (m *Module) Name() string { return "rte" }

const feature = "rte"

func (m *Module) Mount(r chi.Router) {
	h := m.handler
	view := httpx.RequireFeature(feature, "view")
	write := httpx.RequireFeature(feature, "write")

	r.Route("/rte", func(r chi.Router) {
		r.Use(httpx.BlockRoles("teacher"))
		r.With(view).Get("/summary", h.GetSummary)
		r.With(view).Get("/students", h.ListRTEStudents)
		r.Route("/quotas", func(r chi.Router) {
			r.With(view).Get("/", h.ListQuotas)
			r.With(write).Post("/", h.UpsertQuota)
			r.With(write).Delete("/{id}", h.DeleteQuota)
		})
	})
}
