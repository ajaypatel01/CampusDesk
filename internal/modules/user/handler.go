package user

import (
	"encoding/json"
	"net/http"

	"github.com/ajaypatel01/CampusDesk/internal/domain"
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
	u, err := h.svc.Create(r.Context(), in)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, u)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	u, err := h.svc.Get(r.Context(), id)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, u)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	p := pagination.FromRequest(r)
	var schoolID *uuid.UUID
	if sid := r.URL.Query().Get("school_id"); sid != "" {
		id, err := uuid.Parse(sid)
		if err != nil {
			httpx.Error(w, http.StatusBadRequest, "invalid school_id")
			return
		}
		schoolID = &id
	}
	status := domain.UserStatus(r.URL.Query().Get("status"))
	items, total, err := h.svc.List(r.Context(), schoolID, status, p.Limit, p.Offset)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, pagination.NewListResponse(items, total, p.Limit, p.Offset))
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var in LoginInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid json body")
		return
	}
	resp, err := h.svc.Login(r.Context(), in)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// Register handles public self-registration. The created account is inactive
// and pending until an admin approves it via Approve.
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var in RegisterInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid json body")
		return
	}
	u, err := h.svc.Register(r.Context(), in)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, map[string]interface{}{
		"user":    u,
		"message": "registration submitted; an administrator must approve your account before you can log in",
	})
}

// Approve marks a pending user as approved (admin only, see RequireRole in module.go).
func (h *Handler) Approve(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	u, err := h.svc.Approve(r.Context(), id)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, u)
}

// Reject marks a pending user as rejected (admin only, see RequireRole in module.go).
func (h *Handler) Reject(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	u, err := h.svc.Reject(r.Context(), id)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, u)
}
