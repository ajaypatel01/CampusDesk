package fee

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/ajaypatel01/CampusDesk/internal/platform/httpx"
	"github.com/ajaypatel01/CampusDesk/internal/platform/pagination"
	"github.com/ajaypatel01/CampusDesk/internal/platform/whatsapp"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Handler struct {
	svc *Service
	wa  *whatsapp.Client
}

func NewHandler(svc *Service, wa *whatsapp.Client) *Handler {
	return &Handler{svc: svc, wa: wa}
}

// ---- Fee Structures ----

func (h *Handler) CreateFeeStructure(w http.ResponseWriter, r *http.Request) {
	var in CreateFeeStructureInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid json body")
		return
	}
	fs, err := h.svc.CreateFeeStructure(r.Context(), in)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, fs)
}

func (h *Handler) GetFeeStructure(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	fs, err := h.svc.GetFeeStructure(r.Context(), id)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, fs)
}

func (h *Handler) ListFeeStructures(w http.ResponseWriter, r *http.Request) {
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
	items, err := h.svc.ListFeeStructures(r.Context(), schoolID, yearID)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{"items": items})
}

func (h *Handler) UpdateFeeStructure(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	var in UpdateFeeStructureInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid json body")
		return
	}
	fs, err := h.svc.UpdateFeeStructure(r.Context(), id, in)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, fs)
}

// ---- Fee Accounts ----

func (h *Handler) CreateFeeAccount(w http.ResponseWriter, r *http.Request) {
	var in CreateFeeAccountInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid json body")
		return
	}
	fa, err := h.svc.CreateFeeAccount(r.Context(), in)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, fa)
}

func (h *Handler) GetFeeAccount(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	detail, err := h.svc.GetFeeAccount(r.Context(), id)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, detail)
}

func (h *Handler) ListFeeAccounts(w http.ResponseWriter, r *http.Request) {
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
	p := pagination.FromRequest(r)
	f := FeeAccountFilter{
		SchoolID:       schoolID,
		AcademicYearID: yearID,
		Search:         r.URL.Query().Get("search"),
		GradeLevel:     r.URL.Query().Get("grade_level"),
		PaymentStatus:  r.URL.Query().Get("payment_status"),
	}
	items, total, err := h.svc.ListFeeAccounts(r.Context(), f, p.Limit, p.Offset)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, pagination.NewListResponse(items, total, p.Limit, p.Offset))
}

func (h *Handler) UpdateFeeAccount(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	var in UpdateFeeAccountInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid json body")
		return
	}
	fa, err := h.svc.UpdateFeeAccount(r.Context(), id, in)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, fa)
}

// ---- Payments ----

func (h *Handler) RecordPayment(w http.ResponseWriter, r *http.Request) {
	var in RecordPaymentInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid json body")
		return
	}
	p, err := h.svc.RecordPayment(r.Context(), in)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, p)
}

func (h *Handler) ListPayments(w http.ResponseWriter, r *http.Request) {
	accountID, err := uuid.Parse(r.URL.Query().Get("student_fee_account_id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "student_fee_account_id required")
		return
	}
	items, err := h.svc.ListPayments(r.Context(), accountID)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{"items": items})
}

func (h *Handler) VoidPayment(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.svc.VoidPayment(r.Context(), id); err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.NoContent(w)
}

// ---- Summaries ----

func (h *Handler) SchoolFeeSummary(w http.ResponseWriter, r *http.Request) {
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
	summary, err := h.svc.SchoolFeeSummary(r.Context(), schoolID, yearID)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, summary)
}

func (h *Handler) DownloadReceipt(w http.ResponseWriter, r *http.Request) {
	paymentID, err := uuid.Parse(chi.URLParam(r, "payment_id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid payment_id")
		return
	}
	pdfBytes, filename, err := h.svc.GenerateReceipt(r.Context(), paymentID)
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

func (h *Handler) StudentFeeSummary(w http.ResponseWriter, r *http.Request) {
	studentID, err := uuid.Parse(chi.URLParam(r, "student_id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid student_id")
		return
	}
	yearID, err := uuid.Parse(r.URL.Query().Get("academic_year_id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "academic_year_id required")
		return
	}
	summary, err := h.svc.StudentFeeSummary(r.Context(), studentID, yearID)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, summary)
}

func (h *Handler) SendReceiptWhatsApp(w http.ResponseWriter, r *http.Request) {
	if h.wa == nil || !h.wa.Enabled() {
		httpx.Error(w, http.StatusServiceUnavailable, "WhatsApp not configured")
		return
	}
	paymentID, err := uuid.Parse(chi.URLParam(r, "payment_id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid payment_id")
		return
	}
	var body struct {
		Phone string `json:"phone"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Phone == "" {
		httpx.Error(w, http.StatusBadRequest, "phone required")
		return
	}

	pdfBytes, filename, err := h.svc.GenerateReceipt(r.Context(), paymentID)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}

	if err := h.wa.SendDocument(body.Phone, "Fee Receipt", filename, pdfBytes); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to send WhatsApp: "+err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "sent", "phone": body.Phone})
}
