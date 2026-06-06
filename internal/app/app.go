package app

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/ajaypatel01/CampusDesk/internal/config"
	"github.com/ajaypatel01/CampusDesk/internal/modules"
	"github.com/ajaypatel01/CampusDesk/internal/modules/academic"
	"github.com/ajaypatel01/CampusDesk/internal/modules/enrollment"
	"github.com/ajaypatel01/CampusDesk/internal/modules/guardian"
	"github.com/ajaypatel01/CampusDesk/internal/modules/health"
	"github.com/ajaypatel01/CampusDesk/internal/modules/result"
	"github.com/ajaypatel01/CampusDesk/internal/modules/school"
	"github.com/ajaypatel01/CampusDesk/internal/modules/student"
	"github.com/ajaypatel01/CampusDesk/internal/modules/user"
	"github.com/ajaypatel01/CampusDesk/internal/platform/database"
	"github.com/ajaypatel01/CampusDesk/internal/platform/httpx"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v4/pgxpool"
)

type App struct {
	cfg    *config.Config
	pool   *pgxpool.Pool
	server *http.Server
}

func New(ctx context.Context, cfg *config.Config) (*App, error) {
	pool, err := database.NewPool(ctx, cfg.Database.URL)
	if err != nil {
		return nil, err
	}

	router := chi.NewRouter()
	router.Use(httpx.CommonMiddleware()...)
	router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		httpx.JSON(w, http.StatusOK, map[string]string{
			"name":    "CampusDesk API",
			"version": "0.1.0",
		})
	})

	api := chi.NewRouter()
	api.Use(middleware.StripSlashes)
	mountModules(api, pool)
	router.Mount("/api/v1", api)

	srv := &http.Server{
		Addr:         cfg.Addr(),
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	return &App{cfg: cfg, pool: pool, server: srv}, nil
}

func mountModules(r chi.Router, pool *pgxpool.Pool) {
	mods := []modules.Module{
		health.New(pool),
		school.New(pool),
		student.New(pool),
		user.New(pool),
		academic.New(pool),
		enrollment.New(pool),
		guardian.New(pool),
		result.New(pool),
	}
	for _, m := range mods {
		log.Printf("mount module: %s", m.Name())
		m.Mount(r)
	}
}

func (a *App) Run() error {
	log.Printf("CampusDesk API listening on %s", a.cfg.Addr())
	if err := a.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("server: %w", err)
	}
	return nil
}

func (a *App) Shutdown(ctx context.Context) error {
	if err := a.server.Shutdown(ctx); err != nil {
		return err
	}
	a.pool.Close()
	return nil
}
