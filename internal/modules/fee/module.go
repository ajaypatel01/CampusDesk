package fee

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

func (m *Module) Name() string { return "fee" }

func (m *Module) Mount(r chi.Router) {
	h := m.handler

	r.Route("/fee-structures", func(r chi.Router) {
		r.Get("/", h.ListFeeStructures)
		r.Post("/", h.CreateFeeStructure)
		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", h.GetFeeStructure)
			r.Put("/", h.UpdateFeeStructure)
		})
	})

	r.Route("/fee-accounts", func(r chi.Router) {
		r.Get("/", h.ListFeeAccounts)
		r.Post("/", h.CreateFeeAccount)
		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", h.GetFeeAccount)
			r.Put("/", h.UpdateFeeAccount)
		})
	})

	r.Route("/fee-payments", func(r chi.Router) {
		r.Get("/", h.ListPayments)
		r.Post("/", h.RecordPayment)
		r.Delete("/{id}", h.VoidPayment)
	})

	r.Route("/fee-summary", func(r chi.Router) {
		r.Get("/", h.SchoolFeeSummary)
		r.Get("/student/{student_id}", h.StudentFeeSummary)
	})
}
