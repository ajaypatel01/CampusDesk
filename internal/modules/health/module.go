package health

import (
	"context"
	"net/http"

	"github.com/ajaypatel01/CampusDesk/internal/platform/httpx"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v4/pgxpool"
)

type Module struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Module {
	return &Module{pool: pool}
}

func (m *Module) Name() string { return "health" }

func (m *Module) Mount(r chi.Router) {
	r.Get("/health", m.health)
	r.Get("/ready", m.ready)
}

func (m *Module) health(w http.ResponseWriter, _ *http.Request) {
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (m *Module) ready(w http.ResponseWriter, r *http.Request) {
	if err := m.pool.Ping(r.Context()); err != nil {
		httpx.Error(w, http.StatusServiceUnavailable, "database unavailable")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

// Ping exposes DB check for other packages if needed.
func (m *Module) Ping(ctx context.Context) error {
	return m.pool.Ping(ctx)
}
