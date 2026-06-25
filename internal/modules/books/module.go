package books

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

func (m *Module) Name() string { return "books" }

func (m *Module) Mount(r chi.Router) {
	h := m.handler

	r.Route("/books", func(r chi.Router) {
		r.Get("/", h.ListBooks)
		r.Post("/", h.CreateBook)
		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", h.GetBook)
			r.Put("/", h.UpdateBook)
			r.Delete("/", h.DeleteBook)
		})
	})

	r.Route("/book-lists", func(r chi.Router) {
		r.Get("/", h.ListBookLists)
		r.Post("/", h.CreateBookList)
		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", h.GetBookListDetail)
			r.Get("/pdf", h.DownloadBookListPDF)
			r.Post("/items", h.AddItemToList)
			r.Delete("/items/{item_id}", h.RemoveItemFromList)
		})
	})

	r.Route("/book-receipts", func(r chi.Router) {
		r.Get("/", h.ListReceipts)
		r.Post("/", h.RecordReceipt)
	})
}
