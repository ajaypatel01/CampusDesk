package promotion

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

type moveGradeBody struct {
	FromAcademicYearID   string `json:"from_academic_year_id"`
	ToAcademicYearID     string `json:"to_academic_year_id"`
	ToGradeLevelID       string `json:"to_grade_level_id"`
	ClassSectionID       string `json:"class_section_id"`
	CarryForwardDues     bool   `json:"carry_forward_dues"`
	CarryForwardDiscount bool   `json:"carry_forward_discount"`
	CarryForwardVanFee   bool   `json:"carry_forward_van_fee"`
}

func parseMoveGradeBody(r *http.Request) (MoveGradeInput, error) {
	var body moveGradeBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		return MoveGradeInput{}, err
	}
	toYearID, err := uuid.Parse(body.ToAcademicYearID)
	if err != nil {
		return MoveGradeInput{}, err
	}
	toGradeID, err := uuid.Parse(body.ToGradeLevelID)
	if err != nil {
		return MoveGradeInput{}, err
	}
	in := MoveGradeInput{
		ToAcademicYearID: toYearID, ToGradeLevelID: toGradeID,
		CarryForwardDues: body.CarryForwardDues, CarryForwardDiscount: body.CarryForwardDiscount,
		CarryForwardVanFee: body.CarryForwardVanFee,
	}
	if fromYearID, err := uuid.Parse(body.FromAcademicYearID); err == nil {
		in.FromAcademicYearID = fromYearID
	}
	if sectionID, err := uuid.Parse(body.ClassSectionID); err == nil && sectionID != uuid.Nil {
		in.ClassSectionID = &sectionID
	}
	return in, nil
}

// MoveGrade handles a single student's promotion/demotion/class change.
func (h *Handler) MoveGrade(w http.ResponseWriter, r *http.Request) {
	studentID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid student id")
		return
	}
	in, err := parseMoveGradeBody(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid json: to_academic_year_id and to_grade_level_id are required")
		return
	}
	in.StudentID = studentID
	result, err := h.svc.MoveGrade(r.Context(), in)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, result)
}

type bulkMoveBody struct {
	StudentIDs           []string `json:"student_ids"`
	FromAcademicYearID   string   `json:"from_academic_year_id"`
	ToAcademicYearID     string   `json:"to_academic_year_id"`
	ToGradeLevelID       string   `json:"to_grade_level_id"`
	ClassSectionID       string   `json:"class_section_id"`
	CarryForwardDues     bool     `json:"carry_forward_dues"`
	CarryForwardDiscount bool     `json:"carry_forward_discount"`
	CarryForwardVanFee   bool     `json:"carry_forward_van_fee"`
}

// BulkMoveGrade promotes/demotes/reassigns many students in one call --
// the normal shape of an actual end-of-year promotion.
func (h *Handler) BulkMoveGrade(w http.ResponseWriter, r *http.Request) {
	var body bulkMoveBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid json")
		return
	}
	toYearID, err := uuid.Parse(body.ToAcademicYearID)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "to_academic_year_id required")
		return
	}
	toGradeID, err := uuid.Parse(body.ToGradeLevelID)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "to_grade_level_id required")
		return
	}
	if len(body.StudentIDs) == 0 {
		httpx.Error(w, http.StatusBadRequest, "student_ids required")
		return
	}
	in := BulkMoveInput{
		ToAcademicYearID: toYearID, ToGradeLevelID: toGradeID,
		CarryForwardDues: body.CarryForwardDues, CarryForwardDiscount: body.CarryForwardDiscount,
		CarryForwardVanFee: body.CarryForwardVanFee,
	}
	if fromYearID, err := uuid.Parse(body.FromAcademicYearID); err == nil {
		in.FromAcademicYearID = fromYearID
	}
	if sectionID, err := uuid.Parse(body.ClassSectionID); err == nil && sectionID != uuid.Nil {
		in.ClassSectionID = &sectionID
	}
	for _, s := range body.StudentIDs {
		if id, err := uuid.Parse(s); err == nil {
			in.StudentIDs = append(in.StudentIDs, id)
		}
	}
	results := h.svc.BulkMoveGrade(r.Context(), in)
	succeeded := 0
	for _, res := range results {
		if res.Success {
			succeeded++
		}
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{
		"total": len(results), "succeeded": succeeded, "failed": len(results) - succeeded, "results": results,
	})
}

// GetSection returns a student's current class section for an academic year
// (query param academic_year_id), or null if unassigned.
func (h *Handler) GetSection(w http.ResponseWriter, r *http.Request) {
	studentID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid student id")
		return
	}
	yearID, err := uuid.Parse(r.URL.Query().Get("academic_year_id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "academic_year_id required")
		return
	}
	sectionID, err := h.svc.GetSection(r.Context(), studentID, yearID)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{"class_section_id": sectionID})
}

// UpdateSection reassigns one student's class section within an academic year.
func (h *Handler) UpdateSection(w http.ResponseWriter, r *http.Request) {
	studentID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid student id")
		return
	}
	var body struct {
		AcademicYearID string `json:"academic_year_id"`
		ClassSectionID string `json:"class_section_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid json")
		return
	}
	yearID, err := uuid.Parse(body.AcademicYearID)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "academic_year_id required")
		return
	}
	in := UpdateSectionInput{StudentID: studentID, AcademicYearID: yearID}
	if sectionID, err := uuid.Parse(body.ClassSectionID); err == nil && sectionID != uuid.Nil {
		in.ClassSectionID = &sectionID
	}
	if err := h.svc.UpdateSection(r.Context(), in); err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "saved"})
}
