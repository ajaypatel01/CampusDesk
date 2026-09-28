package httpx

import (
	"context"
	"net/http"
)

// FeatureAccess is one user's effective view/write access to one permission
// matrix feature (see internal/domain.Features), after merging any stored
// override on top of the "everything allowed" default.
type FeatureAccess struct {
	View  bool
	Write bool
}

type permsContextKey string

const featurePermsKey permsContextKey = "feature_perms"

// featurePermsState is stored in the request context by whatever loader
// middleware the permissions module registers (see
// internal/modules/permissions.Module.LoaderMiddleware), once per request,
// keyed off the caller's JWT claims.
type featurePermsState struct {
	fullAccess bool
	features   map[string]FeatureAccess
}

// WithFeaturePermissions attaches the caller's effective permission set to
// ctx. fullAccess=true (super_admin, or any request the loader chooses to
// exempt) bypasses every RequireFeature check unconditionally.
func WithFeaturePermissions(ctx context.Context, features map[string]FeatureAccess, fullAccess bool) context.Context {
	return context.WithValue(ctx, featurePermsKey, &featurePermsState{fullAccess: fullAccess, features: features})
}

func featurePermsFromContext(ctx context.Context) *featurePermsState {
	v, _ := ctx.Value(featurePermsKey).(*featurePermsState)
	return v
}

// RequireFeature gates a route on the caller's effective access to a
// permission-matrix feature/action ("view" or "write"). It is layered
// ADDITIONALLY on top of any RequireRole/BlockRoles already guarding a
// route -- it only ever narrows access below what the role check already
// allows, never widens it. With no override on record for a user, every
// feature/action defaults to allowed, so wiring this onto a route changes
// nothing until a super_admin/school_admin actually sets an override for
// that specific user under Settings -> Access Control.
//
// Must run after a loader middleware has called WithFeaturePermissions
// (see internal/modules/permissions); if none ran (e.g. a route mounted
// outside the normal protected group), the check fails open -- unaffected,
// same as before this feature existed -- rather than locking everyone out.
func RequireFeature(feature, action string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			state := featurePermsFromContext(r.Context())
			if state == nil || state.fullAccess {
				next.ServeHTTP(w, r)
				return
			}
			acc, ok := state.features[feature]
			if !ok {
				next.ServeHTTP(w, r) // no override on record for this feature -> default allow
				return
			}
			allowed := acc.View
			if action == "write" {
				allowed = acc.Write
			}
			if !allowed {
				Error(w, http.StatusForbidden, "access denied: your account's access to \""+feature+"\" has been restricted by an administrator")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
