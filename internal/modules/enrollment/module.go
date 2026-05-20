package enrollment

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

func (m *Module) Name() string { return "enrollment" }

func (m *Module) Mount(r chi.Router) {
	h := m.handler
	r.Route("/enrollments", func(r chi.Router) {
		r.Get("/", h.List)
		r.Post("/", h.Create)
		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", h.Get)
			r.Put("/", h.Update)
		})
	})
	r.Route("/attendance", func(r chi.Router) {
		r.Get("/", h.ListAttendance)
		r.Post("/", h.RecordAttendance)
	})
}
