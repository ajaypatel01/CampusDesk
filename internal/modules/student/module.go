package student

import (
	"github.com/ajaypatel01/CampusDesk/internal/modules/guardian"
	"github.com/ajaypatel01/CampusDesk/internal/modules/promotion"
	"github.com/ajaypatel01/CampusDesk/internal/platform/httpx"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v4/pgxpool"
)

type Module struct {
	handler   *Handler
	promotion *promotion.Handler
}

func New(pool *pgxpool.Pool, promotionHandler *promotion.Handler) *Module {
	repo := NewRepository(pool)
	svc := NewService(repo)
	wards := guardian.NewRepository(pool)
	return &Module{handler: NewHandler(svc, wards), promotion: promotionHandler}
}

func (m *Module) Name() string { return "student" }

const feature = "students"

func (m *Module) Mount(r chi.Router) {
	view := httpx.RequireFeature(feature, "view")
	write := httpx.RequireFeature(feature, "write")
	r.With(view).Get("/my-wards", m.handler.MyWards)
	r.Route("/students", func(r chi.Router) {
		// A parent only ever fetches their own ward by id, below — no browsing the roster.
		// A teacher keeps read access (needed to see their own class's
		// roster when entering results) but not the admin write actions
		// below -- creating/editing/deleting student records, bulk import,
		// promotion, or reassigning a section isn't a teaching task.
		r.With(httpx.BlockRoles("parent"), view).Get("/", m.handler.List)
		r.With(httpx.BlockRoles("parent", "teacher"), write).Post("/", m.handler.Create)
		// Bulk import: same access as adding a student one at a time above.
		r.With(httpx.BlockRoles("parent", "teacher"), view).Get("/import-template", m.handler.DownloadImportTemplate)
		r.With(httpx.BlockRoles("parent", "teacher"), write).Post("/import", m.handler.Import)
		// Bulk promotion/demotion (a whole grade or section moving up
		// together at year end) -- same access as everything else here.
		r.With(httpx.BlockRoles("parent", "teacher"), write).Post("/promote-bulk", m.promotion.BulkMoveGrade)
		r.Route("/{id}", func(r chi.Router) {
			r.With(view).Get("/", m.handler.Get)
			r.With(httpx.BlockRoles("parent", "teacher"), write).Put("/", m.handler.Update)
			r.With(httpx.BlockRoles("parent", "teacher"), write).Delete("/", m.handler.Delete)
			// Move to a different grade (promotion/demotion, or a mid-year
			// class change) and reassign class section -- see the
			// promotion package for why these aren't plain field edits.
			r.With(httpx.BlockRoles("parent", "teacher"), write).Post("/move-grade", m.promotion.MoveGrade)
			r.With(view).Get("/section", m.promotion.GetSection)
			r.With(httpx.BlockRoles("parent", "teacher"), write).Put("/section", m.promotion.UpdateSection)
		})
	})
}
