package enrollment

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

func (m *Module) Name() string { return "enrollment" }

const feature = "enrollment"

func (m *Module) Mount(r chi.Router) {
	h := m.handler
	view := httpx.RequireFeature(feature, "view")
	write := httpx.RequireFeature(feature, "write")
	r.Route("/enrollments", func(r chi.Router) {
		r.With(view).Get("/", h.List)
		r.With(write).Post("/", h.Create)
		r.Route("/{id}", func(r chi.Router) {
			r.With(view).Get("/", h.Get)
			r.With(write).Put("/", h.Update)
		})
	})
	r.Route("/attendance", func(r chi.Router) {
		r.With(view).Get("/", h.ListAttendance)
		r.With(write).Post("/", h.RecordAttendance)
	})
}
