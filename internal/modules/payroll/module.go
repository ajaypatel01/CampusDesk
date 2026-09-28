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

// Mount registers the payroll/leave endpoints. Any authenticated role may reach
// these routes; the handlers themselves enforce that only super_admin/school_admin
// can view another staff member's salary or the whole school's payroll - everyone
// else may only view/download their own. Writing leave records (create/update/
// delete) and the whole-school payroll table stay admin-only via this middleware.
const feature = "payroll"

func (m *Module) Mount(r chi.Router) {
	adminOnly := httpx.RequireRole("super_admin", "school_admin")
	view := httpx.RequireFeature(feature, "view")
	write := httpx.RequireFeature(feature, "write")

	r.Route("/staff-leaves", func(r chi.Router) {
		r.With(view).Get("/", m.handler.ListLeaves)
		r.With(adminOnly, write).Post("/", m.handler.CreateLeave)
		r.Route("/{id}", func(r chi.Router) {
			r.With(adminOnly, write).Put("/", m.handler.UpdateLeave)
			r.With(adminOnly, write).Delete("/", m.handler.DeleteLeave)
		})
	})

	r.With(view).Get("/payroll", m.handler.ComputeMonth)
	r.With(view).Get("/payroll/slip", m.handler.DownloadSlip)
}
