package user

import (
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v4/pgxpool"
)

type Module struct {
	handler *Handler
}

func New(pool *pgxpool.Pool, jwtSecret string) *Module {
	repo := NewRepository(pool)
	svc := NewService(repo, jwtSecret)
	return &Module{handler: NewHandler(svc)}
}

func (m *Module) Name() string { return "user" }

// MountPublic registers only the login endpoint (no auth required).
func (m *Module) MountPublic(r chi.Router) {
	r.Post("/auth/login", m.handler.Login)
}

// Mount registers all user management endpoints (auth required).
func (m *Module) Mount(r chi.Router) {
	r.Route("/users", func(r chi.Router) {
		r.Get("/", m.handler.List)
		r.Post("/", m.handler.Create)
		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", m.handler.Get)
		})
	})
}
