package tcvoucher

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/ajaypatel01/CampusDesk/internal/platform/httpx"
	"github.com/ajaypatel01/CampusDesk/internal/platform/pagination"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Handler struct {
	repo *Repository
}

func NewHandler(repo *Repository) *Handler {
	return &Handler{repo: repo}
}

// ---- TC Records ----

func (h *Handler) ListTCRecords(w http.ResponseWriter, r *http.Request) {
	schoolID, err := uuid.Parse(r.URL.Query().Get("school_id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "school_id required")
		return
	}
	p := pagination.FromRequest(r)
	items, total, err := h.repo.ListTCRecords(r.Context(), schoolID, p.Limit, p.Offset)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, pagination.NewListResponse(items, total, p.Limit, p.Offset))
}

func (h *Handler) CreateTCRecord(w http.ResponseWriter, r *http.Request) {
	var t TCRecord
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid json body")
		return
	}
	if t.StudentName == "" {
		httpx.Error(w, http.StatusBadRequest, "student_name required")
		return
	}
	if err := h.repo.CreateTCRecord(r.Context(), &t); err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, t)
}

func (h *Handler) GetTCRecord(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	t, err := h.repo.GetTCRecord(r.Context(), id)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, t)
}

// ---- Vouchers ----

func (h *Handler) ListVouchers(w http.ResponseWriter, r *http.Request) {
	schoolID, err := uuid.Parse(r.URL.Query().Get("school_id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "school_id required")
		return
	}
	p := pagination.FromRequest(r)
	var from, to *time.Time
	if f := r.URL.Query().Get("from"); f != "" {
		t, err := time.Parse("2006-01-02", f)
		if err == nil {
			from = &t
		}
	}
	if f := r.URL.Query().Get("to"); f != "" {
		t, err := time.Parse("2006-01-02", f)
		if err == nil {
			to = &t
		}
	}
	items, total, err := h.repo.ListVouchers(r.Context(), schoolID, from, to, p.Limit, p.Offset)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, pagination.NewListResponse(items, total, p.Limit, p.Offset))
}

func (h *Handler) CreateVoucher(w http.ResponseWriter, r *http.Request) {
	var v Voucher
	if err := json.NewDecoder(r.Body).Decode(&v); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid json body")
		return
	}
	if v.AccountName == "" {
		httpx.Error(w, http.StatusBadRequest, "account_name required")
		return
	}
	if err := h.repo.CreateVoucher(r.Context(), &v); err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, v)
}
