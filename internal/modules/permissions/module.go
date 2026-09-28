package permissions

import (
	"net/http"

	userrepo "github.com/ajaypatel01/CampusDesk/internal/modules/user"
	"github.com/ajaypatel01/CampusDesk/internal/platform/httpx"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v4/pgxpool"
)

type Module struct {
	svc     *Service
	handler *Handler
}

func New(pool *pgxpool.Pool) *Module {
	svc := NewService(NewRepository(pool), userrepo.NewRepository(pool))
	return &Module{svc: svc, handler: NewHandler(svc)}
}

func (m *Module) Name() string { return "permissions" }

// LoaderMiddleware loads the caller's effective feature permissions once per
// request and attaches them to the context, for every later
// httpx.RequireFeature check on that request to read. Must run after
// JWTMiddleware (needs claims) and before any module route is reached. It
// fails open on a DB error -- treating the request as fullAccess -- rather
// than locking every request out if this table has a transient problem;
// the per-route RequireRole/BlockRoles checks already in place are the real
// safety net.
func (m *Module) LoaderMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := httpx.ClaimsFromContext(r.Context())
			features, fullAccess, err := m.svc.EffectiveForRequest(r.Context(), claims)
			if err != nil {
				fullAccess = true
			}
			ctx := httpx.WithFeaturePermissions(r.Context(), features, fullAccess)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// Mount registers the access-control matrix admin API. Only super_admin and
// school_admin may view or change anyone's permissions; Service.canManage
// further restricts a school_admin to their own school's non-admin users.
func (m *Module) Mount(r chi.Router) {
	r.Route("/permissions", func(r chi.Router) {
		r.Use(httpx.RequireRole("super_admin", "school_admin"))
		r.Get("/features", m.handler.ListFeatures)
		r.Route("/users/{userID}", func(r chi.Router) {
			r.Get("/", m.handler.GetMatrix)
			r.Put("/", m.handler.SetMatrix)
		})
	})
}
