package student

import (
	"encoding/json"
	"net/http"

	"github.com/ajaypatel01/CampusDesk/internal/modules/guardian"
	"github.com/ajaypatel01/CampusDesk/internal/platform/httpx"
	"github.com/ajaypatel01/CampusDesk/internal/platform/pagination"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Handler struct {
	svc   *Service
	wards *guardian.Repository
}

func NewHandler(svc *Service, wards *guardian.Repository) *Handler {
	return &Handler{svc: svc, wards: wards}
}

// isWard reports whether the current request's claims belong to a parent whose
// portal access includes studentID. Non-parent roles always return true (unaffected).
func isWard(r *http.Request, wards *guardian.Repository, studentID uuid.UUID) (bool, error) {
	claims := httpx.ClaimsFromContext(r.Context())
	if claims == nil || claims.Role != "parent" {
		return true, nil
	}
	userID, err := uuid.Parse(claims.Sub)
	if err != nil {
		return false, nil
	}
	ids, err := wards.WardStudentIDs(r.Context(), userID)
	if err != nil {
		return false, err
	}
	for _, id := range ids {
		if id == studentID {
			return true, nil
		}
	}
	return false, nil
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

// MyWards returns the student(s) the current logged-in parent has portal access to.
// For any other role this is simply empty (they have no guardian record).
func (h *Handler) MyWards(w http.ResponseWriter, r *http.Request) {
	claims := httpx.ClaimsFromContext(r.Context())
	if claims == nil {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	userID, err := uuid.Parse(claims.Sub)
	if err != nil {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	ids, err := h.wards.WardStudentIDs(r.Context(), userID)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	items := make([]interface{}, 0, len(ids))
	for _, id := range ids {
		st, err := h.svc.Get(r.Context(), id)
		if err != nil {
			continue // skip a ward record that failed to resolve rather than fail the whole list
		}
		items = append(items, st)
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{"items": items})
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	if ok, err := isWard(r, h.wards, id); err != nil {
		httpx.WriteServiceError(w, err)
		return
	} else if !ok {
		httpx.Error(w, http.StatusForbidden, "access denied: not your ward")
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
