package payroll

import (
	"github.com/ajaypatel01/CampusDesk/internal/modules/school"
	"github.com/ajaypatel01/CampusDesk/internal/modules/staff"
	"github.com/ajaypatel01/CampusDesk/internal/platform/httpx"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v4/pgxpool"
)

type Module struct {
	handler *Handler
}

func New(pool *pgxpool.Pool) *Module {
	repo := NewRepository(pool)
	svc := NewService(repo, staff.NewRepository(pool), school.NewRepository(pool))
	return &Module{handler: NewHandler(svc)}
}

func (m *Module) Name() string { return "payroll" }

// Mount registers the payroll/leave endpoints, restricted to super_admin — salary and
// leave data across the whole organization isn't something school_admin/registrar/teacher
// accounts should see.
func (m *Module) Mount(r chi.Router) {
	r.Group(func(r chi.Router) {
		r.Use(httpx.RequireRole("super_admin"))

		r.Route("/staff-leaves", func(r chi.Router) {
			r.Get("/", m.handler.ListLeaves)
			r.Post("/", m.handler.CreateLeave)
			r.Route("/{id}", func(r chi.Router) {
				r.Put("/", m.handler.UpdateLeave)
				r.Delete("/", m.handler.DeleteLeave)
			})
		})

		r.Get("/payroll", m.handler.ComputeMonth)
	})
}
