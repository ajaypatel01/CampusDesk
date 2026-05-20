package academic

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

func (m *Module) Name() string { return "academic" }

func (m *Module) Mount(r chi.Router) {
	h := m.handler
	r.Route("/academic-years", func(r chi.Router) {
		r.Get("/", h.ListYears)
		r.Post("/", h.CreateYear)
	})
	r.Route("/grade-levels", func(r chi.Router) {
		r.Get("/", h.ListGrades)
		r.Post("/", h.CreateGrade)
	})
	r.Route("/class-sections", func(r chi.Router) {
		r.Get("/", h.ListSections)
		r.Post("/", h.CreateSection)
	})
}
