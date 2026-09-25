package student

import (
	"github.com/ajaypatel01/CampusDesk/internal/modules/guardian"
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
	wards := guardian.NewRepository(pool)
	return &Module{handler: NewHandler(svc, wards)}
}

func (m *Module) Name() string { return "student" }

func (m *Module) Mount(r chi.Router) {
	r.Get("/my-wards", m.handler.MyWards)
	r.Route("/students", func(r chi.Router) {
		// A parent only ever fetches their own ward by id, below — no browsing the roster.
		r.With(httpx.BlockRoles("parent")).Get("/", m.handler.List)
		r.With(httpx.BlockRoles("parent")).Post("/", m.handler.Create)
		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", m.handler.Get)
			r.With(httpx.BlockRoles("parent")).Put("/", m.handler.Update)
			r.With(httpx.BlockRoles("parent")).Delete("/", m.handler.Delete)
		})
	})
}
