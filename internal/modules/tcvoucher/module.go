package tcvoucher

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
	return &Module{handler: NewHandler(repo)}
}

func (m *Module) Name() string { return "tcvoucher" }

const feature = "tc_vouchers"

func (m *Module) Mount(r chi.Router) {
	view := httpx.RequireFeature(feature, "view")
	write := httpx.RequireFeature(feature, "write")
	r.Route("/tc-records", func(r chi.Router) {
		r.With(view).Get("/", m.handler.ListTCRecords)
		r.With(write).Post("/", m.handler.CreateTCRecord)
		r.Route("/{id}", func(r chi.Router) {
			r.With(view).Get("/", m.handler.GetTCRecord)
			r.With(write).Put("/", m.handler.UpdateTCRecord)
			r.With(write).Delete("/", m.handler.DeleteTCRecord)
		})
	})
	r.Route("/vouchers", func(r chi.Router) {
		r.With(view).Get("/", m.handler.ListVouchers)
		r.With(write).Post("/", m.handler.CreateVoucher)
	})
}
