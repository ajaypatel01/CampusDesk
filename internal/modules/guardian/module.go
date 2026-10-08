package guardian

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

func (m *Module) Name() string { return "guardian" }

const feature = "guardians"

func (m *Module) Mount(r chi.Router) {
	h := m.handler
	view := httpx.RequireFeature(feature, "view")
	write := httpx.RequireFeature(feature, "write")
	// Guardian/parent contact info management is admin/registrar work, not
	// something a class teacher needs to enter results for their own class.
	r.Route("/guardians", func(r chi.Router) {
		// Guardians are managed by school staff on the student page; parents
		// see their own children through /my-wards instead. Handlers limit
		// everyone but a super_admin to their own school's students.
		r.Use(httpx.BlockRoles("teacher", "parent"))
		r.With(view).Get("/", h.ListByStudent)
		r.With(write).Post("/", h.Create)
		r.With(write).Post("/link", h.Link)
		r.Route("/{id}", func(r chi.Router) {
			r.With(view).Get("/", h.Get)
			r.With(write).Put("/", h.Update)
		})
	})
}
