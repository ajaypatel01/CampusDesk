package permissions

import (
	"encoding/json"
	"net/http"

	"github.com/ajaypatel01/CampusDesk/internal/domain"
	"github.com/ajaypatel01/CampusDesk/internal/platform/httpx"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// ListFeatures returns the fixed feature catalog, for the matrix UI to
// render its rows/labels without hardcoding a second copy.
func (h *Handler) ListFeatures(w http.ResponseWriter, r *http.Request) {
	httpx.JSON(w, http.StatusOK, map[string]interface{}{"items": domain.Features})
}

func (h *Handler) GetMatrix(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(chi.URLParam(r, "userID"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid user id")
		return
	}
	rows, err := h.svc.GetMatrix(r.Context(), httpx.ClaimsFromContext(r.Context()), userID)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{"items": rows})
}

func (h *Handler) SetMatrix(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(chi.URLParam(r, "userID"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid user id")
		return
	}
	var body struct {
		Overrides []SetInput `json:"overrides"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid json body")
		return
	}
	if err := h.svc.SetMatrix(r.Context(), httpx.ClaimsFromContext(r.Context()), userID, body.Overrides); err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	rows, err := h.svc.GetMatrix(r.Context(), httpx.ClaimsFromContext(r.Context()), userID)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{"items": rows})
}
