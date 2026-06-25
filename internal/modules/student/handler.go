package student

import (
	"encoding/json"
	"net/http"

	"github.com/ajaypatel01/CampusDesk/internal/platform/httpx"
	"github.com/ajaypatel01/CampusDesk/internal/platform/pagination"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var in CreateInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid json body")
		return
	}
	st, err := h.svc.Create(r.Context(), in)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, st)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	st, err := h.svc.Get(r.Context(), id)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, st)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	schoolID, err := uuid.Parse(r.URL.Query().Get("school_id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "school_id is required")
		return
	}
	p := pagination.FromRequest(r)
	f := ListFilter{
		SchoolID:      schoolID,
		Status:        r.URL.Query().Get("status"),
		Search:        r.URL.Query().Get("search"),
		Category:      r.URL.Query().Get("category"),
		GradeLevel:    r.URL.Query().Get("grade_level"),
		PaymentStatus: r.URL.Query().Get("payment_status"),
		SortBy:        r.URL.Query().Get("sort_by"),
		SortOrder:     r.URL.Query().Get("sort_order"),
	}
	if ayID := r.URL.Query().Get("academic_year_id"); ayID != "" {
		yearID, err := uuid.Parse(ayID)
		if err != nil {
			httpx.Error(w, http.StatusBadRequest, "invalid academic_year_id")
			return
		}
		f.AcademicYearID = yearID
	}
	items, total, err := h.svc.List(r.Context(), f, p.Limit, p.Offset)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, pagination.NewListResponse(items, total, p.Limit, p.Offset))
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	var in UpdateInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid json body")
		return
	}
	st, err := h.svc.Update(r.Context(), id, in)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, st)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.svc.Delete(r.Context(), id); err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.NoContent(w)
}
