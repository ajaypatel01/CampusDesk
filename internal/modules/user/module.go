package user

import (
	"github.com/ajaypatel01/CampusDesk/internal/platform/httpx"
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

// MountPublic registers the login and self-registration endpoints (no auth required).
func (m *Module) MountPublic(r chi.Router) {
	r.Post("/auth/login", m.handler.Login)
	r.Post("/auth/register", m.handler.Register)
}

// Mount registers all user management endpoints (auth required).
func (m *Module) Mount(r chi.Router) {
	r.Route("/users", func(r chi.Router) {
		r.Use(httpx.BlockRoles("registrar"))
		r.Get("/", m.handler.List)
		r.Post("/", m.handler.Create)
		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", m.handler.Get)
			r.Put("/", m.handler.Update)
			// Approving/rejecting a registration, and deleting an account, are admin-only actions.
			r.Group(func(r chi.Router) {
				r.Use(httpx.RequireRole("super_admin", "school_admin"))
				r.Post("/approve", m.handler.Approve)
				r.Post("/reject", m.handler.Reject)
				r.Delete("/", m.handler.Delete)
			})
		})
	})
}
