package academic

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

func (m *Module) Name() string { return "academic" }

// feature is the access-control-matrix key an admin assigns per-user
// overrides under (Settings -> Access Control); see internal/domain.Features.
const feature = "academic"

func (m *Module) Mount(r chi.Router) {
	h := m.handler
	view := httpx.RequireFeature(feature, "view")
	write := httpx.RequireFeature(feature, "write")
	r.Route("/academic-years", func(r chi.Router) {
		r.With(view).Get("/", h.ListYears)
		r.With(write).Post("/", h.CreateYear)
	})
	r.Route("/grade-levels", func(r chi.Router) {
		r.With(view).Get("/", h.ListGrades)
		// Grade setup, including which report-card template it prints, is
		// admin work, same as subjects/exams in the results module.
		r.With(httpx.BlockRoles("teacher", "parent"), write).Post("/", h.CreateGrade)
		r.With(httpx.BlockRoles("teacher", "parent"), write).Put("/{id}", h.UpdateGrade)
	})
	r.Route("/class-sections", func(r chi.Router) {
		r.With(view).Get("/", h.ListSections)
		r.With(write).Post("/", h.CreateSection)
	})
}
