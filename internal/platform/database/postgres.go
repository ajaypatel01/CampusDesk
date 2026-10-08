package database

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/ajaypatel01/CampusDesk/internal/platform/audit"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

// setActorSQL stores the request's actor in session settings that the
// audit_log trigger reads; resetActorSQL clears them again.
const setActorSQL = `SELECT set_config('app.user_id', $1, false), set_config('app.user_role', $2, false),
	set_config('app.school_id', $3, false), set_config('app.request', $4, false),
	set_config('app.request_id', $5, false), set_config('app.ip', $6, false),
	set_config('app.user_agent', $7, false)`

const resetActorSQL = `SELECT set_config('app.user_id', '', false), set_config('app.user_role', '', false),
	set_config('app.school_id', '', false), set_config('app.request', '', false),
	set_config('app.request_id', '', false), set_config('app.ip', '', false),
	set_config('app.user_agent', '', false)`

func NewPool(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse database url: %w", err)
	}

	// Connections tagged with a request's actor, cleared when released so the
	// next request on the same connection doesn't inherit someone else's name.
	var tagged sync.Map
	cfg.BeforeAcquire = func(ctx context.Context, conn *pgx.Conn) bool {
		a, ok := audit.FromContext(ctx)
		if !ok {
			return true
		}
		if _, err := conn.Exec(ctx, setActorSQL, a.UserID, a.Role, a.SchoolID, a.Request, a.RequestID, a.IP, a.UserAgent); err != nil {
			// The change still goes through; it is just logged without a user.
			log.Printf("audit: tag connection: %v", err)
			return true
		}
		tagged.Store(conn, struct{}{})
		return true
	}
	cfg.AfterRelease = func(conn *pgx.Conn) bool {
		if _, ok := tagged.LoadAndDelete(conn); !ok {
			return true
		}
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if _, err := conn.Exec(ctx, resetActorSQL); err != nil {
			log.Printf("audit: untag connection: %v", err)
			return false // drop the connection rather than reuse it tagged
		}
		return true
	}

	pool, err := pgxpool.ConnectConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("connect database: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return pool, nil
}
