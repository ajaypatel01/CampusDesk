package httpx

import (
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

// SchoolScopeMiddleware ensures school_admin users can only access their own school's data.
// It checks the school_id query param against the school_id in the JWT claims.
func SchoolScopeMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims := ClaimsFromContext(r.Context())
		if claims == nil || claims.Role == "super_admin" || claims.SchoolID == "" {
			next.ServeHTTP(w, r)
			return
		}
		// For school_admin/teacher/registrar/parent: validate school_id param if present
		paramSchoolID := r.URL.Query().Get("school_id")
		if paramSchoolID != "" && paramSchoolID != claims.SchoolID {
			Error(w, http.StatusForbidden, "access denied: school mismatch")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RequireRole restricts a route group to the given JWT roles (e.g. super_admin, school_admin).
// Must run after JWTMiddleware so claims are already in the request context.
func RequireRole(roles ...string) func(http.Handler) http.Handler {
	allowed := make(map[string]bool, len(roles))
	for _, r := range roles {
		allowed[r] = true
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := ClaimsFromContext(r.Context())
			if claims == nil || !allowed[claims.Role] {
				Error(w, http.StatusForbidden, "access denied: insufficient role")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// BlockRoles rejects requests from any of the given JWT roles, leaving every other role unaffected.
// Use this to carve out exceptions (e.g. registrar) from a module that's otherwise open to all
// authenticated roles, without having to enumerate every allowed role.
func BlockRoles(roles ...string) func(http.Handler) http.Handler {
	blocked := make(map[string]bool, len(roles))
	for _, r := range roles {
		blocked[r] = true
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := ClaimsFromContext(r.Context())
			if claims != nil && blocked[claims.Role] {
				Error(w, http.StatusForbidden, "access denied: insufficient role")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func CommonMiddleware() []func(http.Handler) http.Handler {
	return []func(http.Handler) http.Handler{
		middleware.RequestID,
		middleware.RealIP,
		middleware.Recoverer,
		middleware.Timeout(60 * time.Second),
		RequestLogger,
	}
}

func RequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)
		log.Printf("%s %s %d %s", r.Method, r.URL.Path, ww.Status(), time.Since(start))
	})
}
