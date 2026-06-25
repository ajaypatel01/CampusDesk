package van

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

func (m *Module) Name() string { return "van" }

func (m *Module) Mount(r chi.Router) {
	h := m.handler

	r.Route("/vans", func(r chi.Router) {
		r.Get("/", h.ListVans)
		r.Post("/", h.CreateVan)
		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", h.GetVan)
			r.Put("/", h.UpdateVan)
			r.Delete("/", h.DeleteVan)
			r.Post("/routes", h.AddRoute)
			r.Delete("/routes/{route_id}", h.DeleteRoute)
		})
	})

	r.Route("/van-assignments", func(r chi.Router) {
		r.Get("/", h.ListAssignments)
		r.Post("/", h.AssignStudent)
		r.Delete("/{id}", h.RemoveAssignment)
	})
}
