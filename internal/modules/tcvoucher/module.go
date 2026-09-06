package tcvoucher

import (
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

func (m *Module) Mount(r chi.Router) {
	r.Route("/tc-records", func(r chi.Router) {
		r.Get("/", m.handler.ListTCRecords)
		r.Post("/", m.handler.CreateTCRecord)
		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", m.handler.GetTCRecord)
			r.Put("/", m.handler.UpdateTCRecord)
			r.Delete("/", m.handler.DeleteTCRecord)
		})
	})
	r.Route("/vouchers", func(r chi.Router) {
		r.Get("/", m.handler.ListVouchers)
		r.Post("/", m.handler.CreateVoucher)
	})
}
