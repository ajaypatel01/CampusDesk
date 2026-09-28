package permissions

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v4/pgxpool"
)

// Override is one stored row: a user's access to one feature has been
// explicitly set, overriding the "everything allowed" default.
type Override struct {
	FeatureKey string `json:"feature_key"`
	CanView    bool   `json:"can_view"`
	CanWrite   bool   `json:"can_write"`
}

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// ListByUser returns every override on record for userID (no row for a
// feature means "use the default", not "denied").
func (r *Repository) ListByUser(ctx context.Context, userID uuid.UUID) ([]Override, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT feature_key, can_view, can_write FROM permission_overrides WHERE user_id=$1`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Override
	for rows.Next() {
		var o Override
		if err := rows.Scan(&o.FeatureKey, &o.CanView, &o.CanWrite); err != nil {
			return nil, err
		}
		items = append(items, o)
	}
	return items, rows.Err()
}

// ReplaceForUser sets userID's complete override set to exactly overrides --
// any feature previously overridden but not present in overrides reverts to
// the default (its row is deleted), matching how the matrix UI submits the
// full checked/unchecked state of every row at once.
func (r *Repository) ReplaceForUser(ctx context.Context, schoolID *uuid.UUID, userID uuid.UUID, overrides []Override) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `DELETE FROM permission_overrides WHERE user_id=$1`, userID); err != nil {
		return err
	}
	for _, o := range overrides {
		if _, err := tx.Exec(ctx, `
			INSERT INTO permission_overrides (school_id, user_id, feature_key, can_view, can_write)
			VALUES ($1,$2,$3,$4,$5)`,
			schoolID, userID, o.FeatureKey, o.CanView, o.CanWrite,
		); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}
