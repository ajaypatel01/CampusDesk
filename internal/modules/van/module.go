package van

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

func (m *Module) Name() string { return "van" }

const feature = "transport"

func (m *Module) Mount(r chi.Router) {
	h := m.handler
	view := httpx.RequireFeature(feature, "view")
	write := httpx.RequireFeature(feature, "write")

	r.Route("/vans", func(r chi.Router) {
		r.With(view).Get("/", h.ListVans)
		r.With(write).Post("/", h.CreateVan)
		r.Route("/{id}", func(r chi.Router) {
			r.With(view).Get("/", h.GetVan)
			r.With(write).Put("/", h.UpdateVan)
			r.With(write).Delete("/", h.DeleteVan)
			r.With(write).Post("/routes", h.AddRoute)
			r.With(write).Delete("/routes/{route_id}", h.DeleteRoute)
		})
	})

	r.Route("/van-assignments", func(r chi.Router) {
		r.With(view).Get("/", h.ListAssignments)
		r.With(write).Post("/", h.AssignStudent)
		r.With(write).Delete("/{id}", h.RemoveAssignment)
	})
}
