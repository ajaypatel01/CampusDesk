package guardian

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

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var in CreateInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid json body")
		return
	}
	g, err := h.svc.Create(r.Context(), in)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, g)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.svc.CheckGuardian(r.Context(), httpx.ClaimsFromContext(r.Context()), id, false); err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	g, err := h.svc.Get(r.Context(), id)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, g)
}

func (h *Handler) Link(w http.ResponseWriter, r *http.Request) {
	var in LinkInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid json body")
		return
	}
	claims := httpx.ClaimsFromContext(r.Context())
	if err := h.svc.CheckStudent(r.Context(), claims, in.StudentID); err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	if err := h.svc.CheckGuardian(r.Context(), claims, in.GuardianID, true); err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	if err := h.svc.Link(r.Context(), in); err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, httpx.MessageBody{Message: "linked"})
}

func (h *Handler) ListByStudent(w http.ResponseWriter, r *http.Request) {
	studentID, err := uuid.Parse(r.URL.Query().Get("student_id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "student_id required")
		return
	}
	if err := h.svc.CheckStudent(r.Context(), httpx.ClaimsFromContext(r.Context()), studentID); err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	items, err := h.svc.ListByStudent(r.Context(), studentID)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{"items": items})
}

// Update edits a guardian linked to a student in the caller's school.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	var in CreateInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid json body")
		return
	}
	if err := h.svc.CheckGuardian(r.Context(), httpx.ClaimsFromContext(r.Context()), id, false); err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	g, err := h.svc.Update(r.Context(), id, in)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, g)
}
