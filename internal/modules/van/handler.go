package van

import (
	"encoding/json"
	"net/http"

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

func (h *Handler) CreateVan(w http.ResponseWriter, r *http.Request) {
	var in CreateVanInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid json body")
		return
	}
	v, err := h.svc.CreateVan(r.Context(), in)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, v)
}

func (h *Handler) GetVan(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	var yearID uuid.UUID
	yearID, _ = uuid.Parse(r.URL.Query().Get("academic_year_id"))

	detail, err := h.svc.GetVan(r.Context(), id, yearID)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, detail)
}

func (h *Handler) ListVans(w http.ResponseWriter, r *http.Request) {
	schoolID, err := uuid.Parse(r.URL.Query().Get("school_id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "school_id required")
		return
	}
	items, err := h.svc.ListVans(r.Context(), schoolID)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{"items": items})
}

func (h *Handler) UpdateVan(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	var in UpdateVanInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid json body")
		return
	}
	v, err := h.svc.UpdateVan(r.Context(), id, in)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, v)
}

func (h *Handler) DeleteVan(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.svc.DeleteVan(r.Context(), id); err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.NoContent(w)
}

func (h *Handler) AddRoute(w http.ResponseWriter, r *http.Request) {
	vanID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid van id")
		return
	}
	var in AddRouteInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid json body")
		return
	}
	rt, err := h.svc.AddRoute(r.Context(), vanID, in)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, rt)
}

func (h *Handler) DeleteRoute(w http.ResponseWriter, r *http.Request) {
	routeID, err := uuid.Parse(chi.URLParam(r, "route_id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid route_id")
		return
	}
	if err := h.svc.DeleteRoute(r.Context(), routeID); err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.NoContent(w)
}

func (h *Handler) AssignStudent(w http.ResponseWriter, r *http.Request) {
	var in AssignStudentInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid json body")
		return
	}
	a, err := h.svc.AssignStudent(r.Context(), in)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, a)
}

func (h *Handler) RemoveAssignment(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.svc.RemoveAssignment(r.Context(), id); err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.NoContent(w)
}

func (h *Handler) ListAssignments(w http.ResponseWriter, r *http.Request) {
	vanID, err := uuid.Parse(r.URL.Query().Get("van_id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "van_id required")
		return
	}
	yearID, err := uuid.Parse(r.URL.Query().Get("academic_year_id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "academic_year_id required")
		return
	}
	items, err := h.svc.ListAssignments(r.Context(), vanID, yearID)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{"items": items})
}
