package staff

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

func (m *Module) Name() string { return "staff" }

func (m *Module) Mount(r chi.Router) {
	r.Route("/staff", func(r chi.Router) {
		r.Use(httpx.BlockRoles("registrar"))
		r.Get("/", m.handler.List)
		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", m.handler.Get)
			// Editing a staff profile (salary, bank details, CL quota, ...) is
			// admin-only. Every other non-registrar role could reach this
			// endpoint before -- a teacher could edit any other staff member's
			// salary or bank account, not just view it.
			r.With(httpx.RequireRole("super_admin", "school_admin")).Put("/profile", m.handler.UpsertProfile)
		})
	})
}
