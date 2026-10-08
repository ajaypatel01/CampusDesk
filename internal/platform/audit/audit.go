// Package audit carries "who is making this change" from an HTTP request to
// the database connection that runs its queries. The database's audit_log
// trigger (migration 000039) reads it from the app.* session settings.
package audit

import "context"

// Actor describes the request behind a data change.
type Actor struct {
	UserID    string
	Role      string
	SchoolID  string
	Request   string // e.g. "POST /api/v1/exam-marks/bulk"
	RequestID string
	IP        string
	UserAgent string
}

type ctxKey struct{}

// WithActor returns ctx carrying the actor.
func WithActor(ctx context.Context, a Actor) context.Context {
	return context.WithValue(ctx, ctxKey{}, a)
}

// FromContext returns the actor set for this request, if any.
func FromContext(ctx context.Context) (Actor, bool) {
	a, ok := ctx.Value(ctxKey{}).(Actor)
	return a, ok
}
