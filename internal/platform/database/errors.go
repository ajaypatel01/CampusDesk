package database

import (
	"errors"

	apperr "github.com/ajaypatel01/CampusDesk/internal/platform/errors"
	"github.com/jackc/pgconn"
)

func MapError(err error) error {
	if err == nil {
		return nil
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return apperr.ErrConflict
	}
	return err
}
