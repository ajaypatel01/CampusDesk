package payroll

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

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

func (h *Handler) CreateLeave(w http.ResponseWriter, r *http.Request) {
	var in LeaveInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid json body")
		return
	}
	l, err := h.svc.CreateLeave(r.Context(), in)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, l)
}

func (h *Handler) UpdateLeave(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	var in LeaveInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid json body")
		return
	}
	l, err := h.svc.UpdateLeave(r.Context(), id, in)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, l)
}

func (h *Handler) DeleteLeave(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.svc.DeleteLeave(r.Context(), id); err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.NoContent(w)
}

func (h *Handler) ListLeaves(w http.ResponseWriter, r *http.Request) {
	yearID, err := uuid.Parse(r.URL.Query().Get("academic_year_id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "academic_year_id required")
		return
	}
	var items interface{}
	if uid := r.URL.Query().Get("user_id"); uid != "" {
		userID, err := uuid.Parse(uid)
		if err != nil {
			httpx.Error(w, http.StatusBadRequest, "invalid user_id")
			return
		}
		items, err = h.svc.ListLeaves(r.Context(), userID, yearID)
		if err != nil {
			httpx.WriteServiceError(w, err)
			return
		}
	} else {
		schoolID, err := uuid.Parse(r.URL.Query().Get("school_id"))
		if err != nil {
			httpx.Error(w, http.StatusBadRequest, "school_id or user_id required")
			return
		}
		items, err = h.svc.ListLeavesForSchool(r.Context(), schoolID, yearID)
		if err != nil {
			httpx.WriteServiceError(w, err)
			return
		}
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{"items": items})
}

func (h *Handler) ComputeMonth(w http.ResponseWriter, r *http.Request) {
	schoolID, err := uuid.Parse(r.URL.Query().Get("school_id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "school_id required")
		return
	}
	yearID, err := uuid.Parse(r.URL.Query().Get("academic_year_id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "academic_year_id required")
		return
	}
	year, err := strconv.Atoi(r.URL.Query().Get("year"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "year required")
		return
	}
	monthNum, err := strconv.Atoi(r.URL.Query().Get("month"))
	if err != nil || monthNum < 1 || monthNum > 12 {
		httpx.Error(w, http.StatusBadRequest, "month (1-12) required")
		return
	}
	rows, err := h.svc.ComputeMonth(r.Context(), schoolID, yearID, year, time.Month(monthNum))
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{"items": rows})
}

func (h *Handler) DownloadSlip(w http.ResponseWriter, r *http.Request) {
	schoolID, err := uuid.Parse(r.URL.Query().Get("school_id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "school_id required")
		return
	}
	yearID, err := uuid.Parse(r.URL.Query().Get("academic_year_id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "academic_year_id required")
		return
	}
	userID, err := uuid.Parse(r.URL.Query().Get("user_id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "user_id required")
		return
	}
	year, err := strconv.Atoi(r.URL.Query().Get("year"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "year required")
		return
	}
	monthNum, err := strconv.Atoi(r.URL.Query().Get("month"))
	if err != nil || monthNum < 1 || monthNum > 12 {
		httpx.Error(w, http.StatusBadRequest, "month (1-12) required")
		return
	}
	pdfBytes, filename, err := h.svc.GenerateSlip(r.Context(), schoolID, yearID, userID, year, time.Month(monthNum))
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.Header().Set("Content-Length", strconv.Itoa(len(pdfBytes)))
	w.WriteHeader(http.StatusOK)
	w.Write(pdfBytes)
}
