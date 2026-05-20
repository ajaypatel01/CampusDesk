package httpx

import (
	"errors"
	"net/http"

	apperr "github.com/ajaypatel01/CampusDesk/internal/platform/errors"
)

func WriteServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, apperr.ErrNotFound):
		Error(w, http.StatusNotFound, err.Error())
	case errors.Is(err, apperr.ErrConflict):
		Error(w, http.StatusConflict, err.Error())
	case errors.Is(err, apperr.ErrInvalidInput):
		Error(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, apperr.ErrUnauthorized):
		Error(w, http.StatusUnauthorized, err.Error())
	case errors.Is(err, apperr.ErrForbidden):
		Error(w, http.StatusForbidden, err.Error())
	default:
		Error(w, http.StatusInternalServerError, "internal server error")
	}
}
