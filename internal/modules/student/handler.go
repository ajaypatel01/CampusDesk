package student

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

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

// DownloadImportTemplate serves the fillable .xlsx a school fills in and
// uploads back via Import -- see import.go for the exact columns/rules.
func (h *Handler) DownloadImportTemplate(w http.ResponseWriter, r *http.Request) {
	data, err := GenerateImportTemplate()
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "could not generate template")
		return
	}
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", `attachment; filename="student_import_template.xlsx"`)
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

const maxImportFileSize = 10 << 20 // 10MB

// Import accepts an uploaded .xlsx (field name "file") and bulk-creates
// students from it, one row at a time -- see Service.BulkImport for why a
// bad row doesn't block the rest of the batch.
func (h *Handler) Import(w http.ResponseWriter, r *http.Request) {
	schoolID, err := uuid.Parse(r.URL.Query().Get("school_id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "school_id required")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxImportFileSize)
	if err := r.ParseMultipartForm(maxImportFileSize); err != nil {
		httpx.Error(w, http.StatusBadRequest, "file too large (max 10MB) or not a valid upload")
		return
	}
	file, _, err := r.FormFile("file")
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "a \"file\" upload is required")
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "could not read the uploaded file")
		return
	}

	rows, err := ParseImportFile(data)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if len(rows) == 0 {
		httpx.Error(w, http.StatusBadRequest, "no data rows found -- fill in the template's \"Students\" sheet below the header row")
		return
	}

	results := h.svc.BulkImport(r.Context(), schoolID, rows)
	succeeded := 0
	for _, res := range results {
		if res.Success {
			succeeded++
		}
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{
		"total":     len(results),
		"succeeded": succeeded,
		"failed":    len(results) - succeeded,
		"results":   results,
	})
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
	if raw := r.URL.Query().Get("class_section_ids"); raw != "" {
		if f.AcademicYearID == uuid.Nil {
			httpx.Error(w, http.StatusBadRequest, "academic_year_id is required with class_section_ids")
			return
		}
		for _, id := range strings.Split(raw, ",") {
			if _, err := uuid.Parse(id); err != nil {
				httpx.Error(w, http.StatusBadRequest, "invalid class_section_ids")
				return
			}
			f.ClassSectionIDs = append(f.ClassSectionIDs, id)
		}
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
