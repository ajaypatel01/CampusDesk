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

const feature = "students"

func (m *Module) Mount(r chi.Router) {
	view := httpx.RequireFeature(feature, "view")
	write := httpx.RequireFeature(feature, "write")
	r.With(view).Get("/my-wards", m.handler.MyWards)
	r.Route("/students", func(r chi.Router) {
		// A parent only ever fetches their own ward by id, below — no browsing the roster.
		r.With(httpx.BlockRoles("parent"), view).Get("/", m.handler.List)
		r.With(httpx.BlockRoles("parent"), write).Post("/", m.handler.Create)
		// Bulk import: same access as adding a student one at a time above.
		r.With(httpx.BlockRoles("parent"), view).Get("/import-template", m.handler.DownloadImportTemplate)
		r.With(httpx.BlockRoles("parent"), write).Post("/import", m.handler.Import)
		r.Route("/{id}", func(r chi.Router) {
			r.With(view).Get("/", m.handler.Get)
			r.With(httpx.BlockRoles("parent"), write).Put("/", m.handler.Update)
			r.With(httpx.BlockRoles("parent"), write).Delete("/", m.handler.Delete)
		})
	})
}
