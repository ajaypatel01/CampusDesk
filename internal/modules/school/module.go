package school

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

func (m *Module) Name() string { return "school" }

// MountPublic registers the minimal, unauthenticated school directory used by the
// self-registration form (no sensitive fields exposed).
func (m *Module) MountPublic(r chi.Router) {
	r.Get("/schools/public", m.handler.ListPublic)
}

const feature = "school_settings"

func (m *Module) Mount(r chi.Router) {
	view := httpx.RequireFeature(feature, "view")
	write := httpx.RequireFeature(feature, "write")
	r.Route("/schools", func(r chi.Router) {
		r.With(view).Get("/", m.handler.List)
		r.With(write).Post("/", m.handler.Create)
		r.Route("/{id}", func(r chi.Router) {
			r.With(view).Get("/", m.handler.Get)
			r.With(write).Put("/", m.handler.Update)
			r.With(write).Delete("/", m.handler.Delete)
		})
	})
}
