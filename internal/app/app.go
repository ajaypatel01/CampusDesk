package app

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/ajaypatel01/CampusDesk/internal/config"
	"github.com/ajaypatel01/CampusDesk/internal/modules"
	"github.com/ajaypatel01/CampusDesk/internal/modules/academic"
	"github.com/ajaypatel01/CampusDesk/internal/modules/auditlog"
	"github.com/ajaypatel01/CampusDesk/internal/modules/books"
	"github.com/ajaypatel01/CampusDesk/internal/modules/communications"
	"github.com/ajaypatel01/CampusDesk/internal/modules/customfields"
	"github.com/ajaypatel01/CampusDesk/internal/modules/documents"
	"github.com/ajaypatel01/CampusDesk/internal/modules/enrollment"
	"github.com/ajaypatel01/CampusDesk/internal/modules/fee"
	"github.com/ajaypatel01/CampusDesk/internal/modules/guardian"
	"github.com/ajaypatel01/CampusDesk/internal/modules/health"
	"github.com/ajaypatel01/CampusDesk/internal/modules/homework"
	"github.com/ajaypatel01/CampusDesk/internal/modules/idcard"
	"github.com/ajaypatel01/CampusDesk/internal/modules/media"
	"github.com/ajaypatel01/CampusDesk/internal/modules/payroll"
	"github.com/ajaypatel01/CampusDesk/internal/modules/permissions"
	"github.com/ajaypatel01/CampusDesk/internal/modules/promotion"
	"github.com/ajaypatel01/CampusDesk/internal/modules/results"
	"github.com/ajaypatel01/CampusDesk/internal/modules/rte"
	"github.com/ajaypatel01/CampusDesk/internal/modules/school"
	"github.com/ajaypatel01/CampusDesk/internal/modules/staff"
	"github.com/ajaypatel01/CampusDesk/internal/modules/student"
	"github.com/ajaypatel01/CampusDesk/internal/modules/tcvoucher"
	"github.com/ajaypatel01/CampusDesk/internal/modules/user"
	"github.com/ajaypatel01/CampusDesk/internal/modules/van"
	"github.com/ajaypatel01/CampusDesk/internal/platform/database"
	"github.com/ajaypatel01/CampusDesk/internal/platform/email"
	"github.com/ajaypatel01/CampusDesk/internal/platform/fast2sms"
	"github.com/ajaypatel01/CampusDesk/internal/platform/httpx"
	"github.com/ajaypatel01/CampusDesk/internal/platform/storage"
	"github.com/ajaypatel01/CampusDesk/internal/platform/whatsapp"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/google/uuid"
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

	emailClient := email.New(cfg.Email.ResendAPIKey, cfg.Email.FromEmail, cfg.Email.FromName)

	storageClient, err := storage.New(storage.Config{
		Endpoint:        cfg.Storage.Endpoint,
		Region:          cfg.Storage.Region,
		Bucket:          cfg.Storage.Bucket,
		AccessKeyID:     cfg.Storage.AccessKeyID,
		SecretAccessKey: cfg.Storage.SecretAccessKey,
		UseSSL:          cfg.Storage.UseSSL,
	})
	if err != nil {
		log.Printf("warn: storage client init failed: %v — photo/ID-card features disabled", err)
		storageClient = nil
	}

	waClient := whatsapp.New(cfg.WhatsApp.PhoneNumberID, cfg.WhatsApp.AccessToken, cfg.WhatsApp.APIVersion).
		WithBaseURL(cfg.WhatsApp.BaseURL)

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

	otpSenders := map[uuid.UUID]string{}
	for school, numberID := range cfg.WhatsApp.OTPSchoolSenders {
		id, err := uuid.Parse(school)
		if err != nil {
			log.Printf("warn: WHATSAPP_OTP_SENDERS: %q is not a school id, skipped", school)
			continue
		}
		otpSenders[id] = numberID
	}
	userMod := user.New(pool, cfg.Auth.JWTSecret, emailClient, user.OTPSender{
		Client: waClient, Template: cfg.WhatsApp.OTPTemplate, Language: cfg.WhatsApp.OTPLanguage, SchoolSenders: otpSenders,
		SMS: fast2sms.New(cfg.Fast2SMSAPIKey, cfg.Fast2SMSURL),
	}, cfg.FrontendURL)
	schoolMod := school.New(pool)
	permsMod := permissions.New(pool)

	// Public routes (no auth required)
	api.Group(func(r chi.Router) {
		r.Use(httpx.AuditActor) // records public writes (e.g. registration) without a user
		health.New(pool).Mount(r)
		userMod.MountPublic(r)   // /auth/login, /auth/register
		schoolMod.MountPublic(r) // /schools/public — school picker for registration
	})

	// Protected routes — JWT required
	api.Group(func(r chi.Router) {
		r.Use(httpx.JWTMiddleware(cfg.Auth.JWTSecret))
		// Tags each write request with the logged-in user for the audit log.
		r.Use(httpx.AuditActor)
		// Rejects any token minted before the user's last "log out
		// everywhere" -- must run right after JWT parsing, before anything
		// else trusts the claims.
		r.Use(userMod.EnforceTokenVersion())
		r.Use(httpx.SchoolScopeMiddleware)
		// Loads each caller's access-control-matrix overrides once per request
		// so every module's httpx.RequireFeature checks below can read them
		// from context instead of each querying the DB themselves.
		r.Use(permsMod.LoaderMiddleware())
		// expose feature flags so frontend can adapt UI
		r.Get("/config", func(w http.ResponseWriter, r *http.Request) {
			httpx.JSON(w, http.StatusOK, map[string]bool{
				"whatsapp_enabled": waClient.Enabled(),
			})
		})
		userMod.Mount(r)  // /users CRUD
		permsMod.Mount(r) // /permissions/* -- access-control matrix admin API
		mountProtectedModules(r, schoolMod, pool, emailClient, storageClient, waClient, cfg.Auth.JWTSecret)
	})

	router.Mount("/api/v1", api)

	srv := &http.Server{
		Addr:         cfg.Addr(),
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	return &App{cfg: cfg, pool: pool, server: srv}, nil
}

func mountProtectedModules(r chi.Router, schoolMod *school.Module, pool *pgxpool.Pool, emailClient *email.Client, storageClient *storage.Client, waClient *whatsapp.Client, jwtSecret string) {
	// Promotion composes student/academic/fee/enrollment repositories
	// directly (rather than being its own top-level module) so its routes
	// can be mounted inside the student module's existing route tree --
	// see internal/modules/promotion's doc comment for why moving grade is
	// really a fee-account operation, not a plain student field edit.
	promotionHandler := promotion.NewHandler(promotion.NewService(
		student.NewRepository(pool), academic.NewRepository(pool), fee.NewRepository(pool), enrollment.NewRepository(pool),
	))

	mods := []modules.Module{
		schoolMod,
		student.New(pool, promotionHandler),
		academic.New(pool),
		enrollment.New(pool),
		guardian.New(pool),
		fee.New(pool, waClient),
		documents.New(pool, emailClient, waClient),
		van.New(pool),
		rte.New(pool),
		books.New(pool),
		media.New(pool, storageClient),
		idcard.New(pool, storageClient),
		results.New(pool),
		homework.New(pool),
		communications.New(pool, waClient),
		staff.New(pool),
		tcvoucher.New(pool),
		payroll.New(pool),
		customfields.New(pool),
		auditlog.New(pool),
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
