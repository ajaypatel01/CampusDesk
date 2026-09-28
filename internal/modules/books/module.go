package books

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

func (m *Module) Name() string { return "books" }

const feature = "books"

func (m *Module) Mount(r chi.Router) {
	h := m.handler
	view := httpx.RequireFeature(feature, "view")
	write := httpx.RequireFeature(feature, "write")

	r.Group(func(r chi.Router) {
		r.Use(httpx.BlockRoles("registrar"))

		r.Route("/books", func(r chi.Router) {
			r.With(view).Get("/", h.ListBooks)
			r.With(write).Post("/", h.CreateBook)
			r.Route("/{id}", func(r chi.Router) {
				r.With(view).Get("/", h.GetBook)
				r.With(write).Put("/", h.UpdateBook)
				r.With(write).Delete("/", h.DeleteBook)
			})
		})

		r.Route("/book-lists", func(r chi.Router) {
			r.With(view).Get("/", h.ListBookLists)
			r.With(write).Post("/", h.CreateBookList)
			r.Route("/{id}", func(r chi.Router) {
				r.With(view).Get("/", h.GetBookListDetail)
				r.With(view).Get("/pdf", h.DownloadBookListPDF)
				r.With(write).Post("/items", h.AddItemToList)
				r.With(write).Delete("/items/{item_id}", h.RemoveItemFromList)
			})
		})

		r.Route("/book-receipts", func(r chi.Router) {
			r.With(view).Get("/", h.ListReceipts)
			r.With(write).Post("/", h.RecordReceipt)
		})
	})
}
