package user

import (
	"net/http"

	"github.com/ajaypatel01/CampusDesk/internal/platform/email"
	"github.com/ajaypatel01/CampusDesk/internal/platform/httpx"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v4/pgxpool"
)

type Module struct {
	repo    *Repository
	handler *Handler
}

func New(pool *pgxpool.Pool, jwtSecret string, emailClient *email.Client, otp OTPSender, frontendURL string) *Module {
	repo := NewRepository(pool)
	svc := NewService(repo, jwtSecret, emailClient, otp, frontendURL)
	return &Module{repo: repo, handler: NewHandler(svc)}
}

// EnforceTokenVersion 401s any request whose JWT was minted before the
// user's last "log out everywhere" (see Service.LogoutEverywhere) --
// checked by comparing the token's "tv" claim against the user's current
// token_version. Must run after httpx.JWTMiddleware, early in the protected
// route group, before anything else trusts the claims. Fails open on a
// transient DB error or a missing/malformed claim -- same reasoning as the
// permissions loader middleware: don't lock every request out over a blip,
// the normal role/permission checks are still the real gate.
func (m *Module) EnforceTokenVersion() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := httpx.ClaimsFromContext(r.Context())
			if claims == nil {
				next.ServeHTTP(w, r)
				return
			}
			userID, err := uuid.Parse(claims.Sub)
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}
			current, err := m.repo.GetTokenVersion(r.Context(), userID)
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}
			if current != claims.TokenVersion {
				httpx.Error(w, http.StatusUnauthorized, "session revoked -- please log in again")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func (m *Module) Name() string { return "user" }

// MountPublic registers the login and self-registration endpoints (no auth required).
func (m *Module) MountPublic(r chi.Router) {
	r.Post("/auth/login", m.handler.Login)
	r.Post("/auth/register", m.handler.Register)
	r.Post("/auth/password-reset/request", m.handler.RequestPasswordReset)
	r.Post("/auth/password-reset/confirm", m.handler.ConfirmPasswordReset)
	// WhatsApp OTP login and phone-based password reset -- both return a
	// clear "not configured" error until WhatsApp is set up (see otp.go).
	r.Post("/auth/otp/send", m.handler.RequestOTPLogin)
	r.Post("/auth/otp/verify", m.handler.VerifyOTPLogin)
	r.Post("/auth/password-reset/otp-request", m.handler.RequestPasswordResetOTP)
	r.Post("/auth/password-reset/otp-confirm", m.handler.ConfirmPasswordResetOTP)
}

const feature = "user_management"

// Mount registers all user management endpoints (auth required).
func (m *Module) Mount(r chi.Router) {
	view := httpx.RequireFeature(feature, "view")
	write := httpx.RequireFeature(feature, "write")
	// Any authenticated user may log themselves out everywhere, fetch their
	// own profile, or verify their own phone number -- none of these are
	// gated by the user_management feature, which is about administering
	// OTHER accounts. /auth/me exists so every role (including registrar,
	// blocked from /users/{id} below) can read their own record.
	r.Post("/auth/logout-everywhere", m.handler.LogoutEverywhere)
	r.Post("/auth/password/change", m.handler.ChangePassword)
	r.Post("/auth/email/verify/request", m.handler.RequestEmailVerification)
	r.Post("/auth/email/verify/confirm", m.handler.ConfirmEmailVerification)
	r.Get("/auth/me", m.handler.Me)
	r.Post("/auth/phone/verify/request", m.handler.RequestPhoneVerification)
	r.Post("/auth/phone/verify/confirm", m.handler.ConfirmPhoneVerification)
	r.Route("/users", func(r chi.Router) {
		// Browsing/editing other accounts is admin work -- everyone else's own
		// profile comes from /auth/me above. The handlers further limit a
		// school_admin to their own school's non-super_admin users.
		r.Use(httpx.RequireRole("super_admin", "school_admin"))
		r.With(view).Get("/", m.handler.List)
		r.With(write).Post("/", m.handler.Create)
		r.Route("/{id}", func(r chi.Router) {
			r.With(view).Get("/", m.handler.Get)
			r.With(write).Put("/", m.handler.Update)
			r.Group(func(r chi.Router) {
				r.Use(write)
				r.Post("/approve", m.handler.Approve)
				r.Post("/reject", m.handler.Reject)
				r.Delete("/", m.handler.Delete)
			})
		})
	})
}
