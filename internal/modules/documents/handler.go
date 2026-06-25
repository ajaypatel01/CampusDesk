package documents

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/ajaypatel01/CampusDesk/internal/platform/email"
	"github.com/ajaypatel01/CampusDesk/internal/platform/httpx"
	"github.com/ajaypatel01/CampusDesk/internal/platform/whatsapp"
	"github.com/google/uuid"
)

type Handler struct {
	svc         *Service
	emailClient *email.Client
	wa          *whatsapp.Client
}

func NewHandler(svc *Service, emailClient *email.Client, wa *whatsapp.Client) *Handler {
	return &Handler{svc: svc, emailClient: emailClient, wa: wa}
}

func (h *Handler) DownloadBonafide(w http.ResponseWriter, r *http.Request) {
	studentID, err := uuid.Parse(r.URL.Query().Get("student_id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "student_id required")
		return
	}
	yearID, err := uuid.Parse(r.URL.Query().Get("academic_year_id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "academic_year_id required")
		return
	}
	pdfBytes, filename, err := h.svc.GenerateBonafide(r.Context(), studentID, yearID)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	writePDF(w, pdfBytes, filename)
}

func (h *Handler) DownloadTC(w http.ResponseWriter, r *http.Request) {
	studentID, err := uuid.Parse(r.URL.Query().Get("student_id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "student_id required")
		return
	}

	var dateOfLeaving time.Time
	if s := r.URL.Query().Get("date_of_leaving"); s != "" {
		for _, layout := range []string{"2006-01-02", "02/01/2006"} {
			if t, err := time.Parse(layout, s); err == nil {
				dateOfLeaving = t
				break
			}
		}
	}

	reason := r.URL.Query().Get("reason")
	conduct := r.URL.Query().Get("conduct")

	pdfBytes, filename, err := h.svc.GenerateTC(r.Context(), studentID, dateOfLeaving, reason, conduct)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	writePDF(w, pdfBytes, filename)
}

func (h *Handler) DownloadSalarySlip(w http.ResponseWriter, r *http.Request) {
	var in SalarySlipInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid json body")
		return
	}
	pdfBytes, filename, err := h.svc.GenerateSalarySlip(r.Context(), in)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	writePDF(w, pdfBytes, filename)
}

func writePDF(w http.ResponseWriter, data []byte, filename string) {
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

// ---- Email handlers ----

type emailRequest struct {
	RecipientEmail string `json:"recipient_email"`
	RecipientName  string `json:"recipient_name"`
}

func (h *Handler) EmailBonafide(w http.ResponseWriter, r *http.Request) {
	var body struct {
		StudentID      string `json:"student_id"`
		AcademicYearID string `json:"academic_year_id"`
		emailRequest
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid json body")
		return
	}
	studentID, err := uuid.Parse(body.StudentID)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "student_id required")
		return
	}
	yearID, err := uuid.Parse(body.AcademicYearID)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "academic_year_id required")
		return
	}
	if body.RecipientEmail == "" {
		httpx.Error(w, http.StatusBadRequest, "recipient_email required")
		return
	}
	pdfBytes, filename, err := h.svc.GenerateBonafide(r.Context(), studentID, yearID)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	if err := h.emailClient.SendPDF(
		body.RecipientEmail, body.RecipientName,
		"Bonafide Certificate",
		"<p>Please find the attached Bonafide Certificate.</p>",
		filename, pdfBytes,
	); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to send email: "+err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"message": "email sent successfully"})
}

func (h *Handler) EmailTC(w http.ResponseWriter, r *http.Request) {
	var body struct {
		StudentID     string `json:"student_id"`
		DateOfLeaving string `json:"date_of_leaving"`
		Reason        string `json:"reason"`
		Conduct       string `json:"conduct"`
		emailRequest
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid json body")
		return
	}
	studentID, err := uuid.Parse(body.StudentID)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "student_id required")
		return
	}
	if body.RecipientEmail == "" {
		httpx.Error(w, http.StatusBadRequest, "recipient_email required")
		return
	}
	var dateOfLeaving time.Time
	if body.DateOfLeaving != "" {
		for _, layout := range []string{"2006-01-02", "02/01/2006"} {
			if t, err := time.Parse(layout, body.DateOfLeaving); err == nil {
				dateOfLeaving = t
				break
			}
		}
	}
	pdfBytes, filename, err := h.svc.GenerateTC(r.Context(), studentID, dateOfLeaving, body.Reason, body.Conduct)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	if err := h.emailClient.SendPDF(
		body.RecipientEmail, body.RecipientName,
		"Transfer Certificate",
		"<p>Please find the attached Transfer Certificate.</p>",
		filename, pdfBytes,
	); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to send email: "+err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"message": "email sent successfully"})
}

func (h *Handler) EmailSalarySlip(w http.ResponseWriter, r *http.Request) {
	var body struct {
		SalarySlipInput
		emailRequest
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid json body")
		return
	}
	if body.RecipientEmail == "" {
		httpx.Error(w, http.StatusBadRequest, "recipient_email required")
		return
	}
	pdfBytes, filename, err := h.svc.GenerateSalarySlip(r.Context(), body.SalarySlipInput)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	subject := fmt.Sprintf("Salary Slip - %s %d", body.Month, body.Year)
	body2 := fmt.Sprintf("<p>Dear %s,</p><p>Please find your salary slip for %s %d attached.</p>",
		body.RecipientName, body.Month, body.Year)
	if err := h.emailClient.SendPDF(
		body.RecipientEmail, body.RecipientName,
		subject, body2, filename, pdfBytes,
	); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to send email: "+err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"message": "email sent successfully"})
}

// ---- WhatsApp handlers ----

type waPhoneRequest struct {
	Phone string `json:"phone"`
}

func (h *Handler) WhatsAppBonafide(w http.ResponseWriter, r *http.Request) {
	if !h.wa.Enabled() {
		httpx.Error(w, http.StatusServiceUnavailable, "WhatsApp not configured")
		return
	}
	var body struct {
		StudentID      string `json:"student_id"`
		AcademicYearID string `json:"academic_year_id"`
		waPhoneRequest
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid json body")
		return
	}
	studentID, err := uuid.Parse(body.StudentID)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "student_id required")
		return
	}
	yearID, err := uuid.Parse(body.AcademicYearID)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "academic_year_id required")
		return
	}
	if body.Phone == "" {
		httpx.Error(w, http.StatusBadRequest, "phone required")
		return
	}
	pdfBytes, filename, err := h.svc.GenerateBonafide(r.Context(), studentID, yearID)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	if err := h.wa.SendDocument(body.Phone, "Bonafide Certificate", filename, pdfBytes); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "whatsapp send failed: "+err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "sent", "phone": body.Phone})
}

func (h *Handler) WhatsAppTC(w http.ResponseWriter, r *http.Request) {
	if !h.wa.Enabled() {
		httpx.Error(w, http.StatusServiceUnavailable, "WhatsApp not configured")
		return
	}
	var body struct {
		StudentID     string `json:"student_id"`
		DateOfLeaving string `json:"date_of_leaving"`
		Reason        string `json:"reason"`
		Conduct       string `json:"conduct"`
		waPhoneRequest
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid json body")
		return
	}
	studentID, err := uuid.Parse(body.StudentID)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "student_id required")
		return
	}
	if body.Phone == "" {
		httpx.Error(w, http.StatusBadRequest, "phone required")
		return
	}
	var dateOfLeaving time.Time
	if body.DateOfLeaving != "" {
		for _, layout := range []string{"2006-01-02", "02/01/2006"} {
			if t, err2 := time.Parse(layout, body.DateOfLeaving); err2 == nil {
				dateOfLeaving = t
				break
			}
		}
	}
	pdfBytes, filename, err := h.svc.GenerateTC(r.Context(), studentID, dateOfLeaving, body.Reason, body.Conduct)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	if err := h.wa.SendDocument(body.Phone, "Transfer Certificate", filename, pdfBytes); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "whatsapp send failed: "+err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "sent", "phone": body.Phone})
}

func (h *Handler) WhatsAppSalarySlip(w http.ResponseWriter, r *http.Request) {
	if !h.wa.Enabled() {
		httpx.Error(w, http.StatusServiceUnavailable, "WhatsApp not configured")
		return
	}
	var body struct {
		SalarySlipInput
		waPhoneRequest
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid json body")
		return
	}
	if body.Phone == "" {
		httpx.Error(w, http.StatusBadRequest, "phone required")
		return
	}
	pdfBytes, filename, err := h.svc.GenerateSalarySlip(r.Context(), body.SalarySlipInput)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	caption := fmt.Sprintf("Salary Slip - %s %d", body.Month, body.Year)
	if err := h.wa.SendDocument(body.Phone, caption, filename, pdfBytes); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "whatsapp send failed: "+err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "sent", "phone": body.Phone})
}
