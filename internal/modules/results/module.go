package results

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ajaypatel01/CampusDesk/internal/platform/archive"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ajaypatel01/CampusDesk/internal/domain"
	"github.com/ajaypatel01/CampusDesk/internal/modules/guardian"
	apperr "github.com/ajaypatel01/CampusDesk/internal/platform/errors"
	"github.com/ajaypatel01/CampusDesk/internal/platform/httpx"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v4/pgxpool"
)

type Module struct {
	repo    *Repository
	wards   *guardian.Repository
	archive *archive.Archiver
}

func New(pool *pgxpool.Pool, arch *archive.Archiver) *Module {
	return &Module{repo: NewRepository(pool), wards: guardian.NewRepository(pool), archive: arch}
}

func (m *Module) Name() string { return "results" }

const feature = "results"

func (m *Module) Mount(r chi.Router) {
	view := httpx.RequireFeature(feature, "view")
	write := httpx.RequireFeature(feature, "write")
	r.Route("/subjects", func(r chi.Router) {
		r.With(httpx.BlockRoles("parent"), view).Get("/", m.ListSubjects)
		// Subject/exam setup is admin work; class teachers only view and enter marks.
		// Parents never manage results data, only view their own ward's marksheet below.
		r.With(httpx.BlockRoles("teacher", "parent"), write).Post("/", m.CreateSubject)
		r.With(httpx.BlockRoles("teacher", "parent"), write).Put("/{id}", m.UpdateSubject)
		r.With(httpx.BlockRoles("teacher", "parent"), write).Delete("/{id}", m.DeleteSubject)
		// Adding/editing/removing a graded component ("section") changes how a subject
		// is marked, so it's subject setup like the routes above -- teachers
		// can list components (to enter marks against them) but not change them.
		r.Route("/{id}/mark-components", func(r chi.Router) {
			r.With(httpx.BlockRoles("parent"), view).Get("/", m.ListSubjectComponents)
			r.With(httpx.BlockRoles("teacher", "parent"), write).Post("/", m.AddSubjectComponent)
			r.With(httpx.BlockRoles("teacher", "parent"), write).Put("/{key}", m.UpdateSubjectComponent)
			r.With(httpx.BlockRoles("teacher", "parent"), write).Delete("/{key}", m.DeleteSubjectComponent)
		})
	})
	r.Route("/exams", func(r chi.Router) {
		r.With(httpx.BlockRoles("parent"), view).Get("/", m.ListExams)
		r.With(httpx.BlockRoles("teacher", "parent"), write).Post("/", m.CreateExam)
		r.With(httpx.BlockRoles("teacher", "parent"), write).Put("/{id}", m.UpdateExam)
		r.With(httpx.BlockRoles("teacher", "parent"), write).Delete("/{id}", m.DeleteExam)
		r.With(httpx.BlockRoles("teacher", "parent"), write).Post("/{id}/publish", m.PublishExam)
		// Whole-class result sheet for one exam: admins and the owner only.
		r.With(httpx.RequireRole("super_admin", "school_admin", "registrar", "teacher"), view).Get("/{id}/result-sheet", m.GetResultSheet)
		// Per-exam marks distribution for each subject. Teachers read it (to
		// enter marks against it) for their own class; only admins change it.
		r.With(httpx.BlockRoles("parent"), view).Get("/{id}/mark-formats", m.ListExamFormats)
		r.With(httpx.BlockRoles("teacher", "parent"), write).Put("/{id}/mark-formats/{subjectId}", m.SetExamSubjectFormat)
		r.With(httpx.BlockRoles("teacher", "parent"), write).Delete("/{id}/mark-formats/{subjectId}", m.ResetExamSubjectFormat)
	})
	r.Route("/exam-marks", func(r chi.Router) {
		// One student's saved marks in an exam, to pre-fill the entry form.
		r.With(httpx.BlockRoles("parent"), view).Get("/", m.ListStudentExamMarks)
		r.With(httpx.BlockRoles("parent"), write).Post("/", m.UpsertMark)
		r.With(httpx.BlockRoles("parent"), write).Post("/bulk", m.BulkUpsertMarks)
	})
	r.With(view).Get("/ward-exams", m.WardExams)
	r.Route("/marksheets", func(r chi.Router) {
		r.With(view).Get("/", m.GetMarksheet)
		r.With(view).Get("/pdf", m.DownloadMarksheet)
		// Manually correcting a marksheet's total is a super_admin-only override,
		// not a general "edit results" permission -- registrars/school admins
		// still only enter marks through the normal per-subject flow above.
		r.With(httpx.RequireRole("super_admin"), write).Put("/total-override", m.SetTotalOverride)
		r.With(httpx.RequireRole("super_admin"), write).Delete("/total-override", m.DeleteTotalOverride)
	})
	r.Route("/report-cards", func(r chi.Router) {
		r.With(view).Get("/", m.GetReportCard)
		r.With(view).Get("/pdf", m.DownloadReportCard)
		// Attendance/remark/promoted-to and discipline grades are filled in
		// by the class teacher, same access as entering marks -- open to
		// teacher/admin, never parent.
		r.With(httpx.BlockRoles("parent"), write).Put("/details", m.UpsertReportCardDetails)
		r.With(httpx.BlockRoles("parent"), write).Put("/discipline-grades", m.UpsertDisciplineGrades)
	})
	r.With(view).Get("/discipline-criteria", m.ListDisciplineCriteria)
}

// ---- Subject handlers ----

func (m *Module) CreateSubject(w http.ResponseWriter, r *http.Request) {
	var in struct {
		SchoolID       string `json:"school_id"`
		GradeLevelID   string `json:"grade_level_id"`
		Name           string `json:"name"`
		Code           string `json:"code"`
		MaxMarks       int    `json:"max_marks"`
		PassingMarks   int    `json:"passing_marks"`
		SortOrder      int    `json:"sort_order"`
		IsCoScholastic bool   `json:"is_co_scholastic"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid json")
		return
	}
	schoolID, _ := uuid.Parse(in.SchoolID)
	gradeID, _ := uuid.Parse(in.GradeLevelID)
	if schoolID == uuid.Nil || gradeID == uuid.Nil || in.Name == "" {
		httpx.Error(w, http.StatusBadRequest, "school_id, grade_level_id, name required")
		return
	}
	if in.MaxMarks <= 0 {
		in.MaxMarks = 100
	}
	if in.PassingMarks <= 0 {
		in.PassingMarks = 33
	}
	s := &domain.Subject{
		SchoolID: schoolID, GradeLevelID: gradeID, Name: in.Name, Code: in.Code,
		MaxMarks: in.MaxMarks, PassingMarks: in.PassingMarks, SortOrder: in.SortOrder,
		IsCoScholastic: in.IsCoScholastic,
	}
	if err := m.repo.CreateSubject(r.Context(), s); err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, s)
}

func (m *Module) ListSubjects(w http.ResponseWriter, r *http.Request) {
	schoolID, _ := uuid.Parse(r.URL.Query().Get("school_id"))
	gradeID, _ := uuid.Parse(r.URL.Query().Get("grade_level_id"))
	if schoolID == uuid.Nil || gradeID == uuid.Nil {
		httpx.Error(w, http.StatusBadRequest, "school_id and grade_level_id required")
		return
	}
	if ok, err := m.teacherCanAccessGrade(r.Context(), httpx.ClaimsFromContext(r.Context()), gradeID); err != nil {
		httpx.WriteServiceError(w, err)
		return
	} else if !ok {
		httpx.Error(w, http.StatusForbidden, "access denied: not your class")
		return
	}
	items, err := m.repo.ListSubjects(r.Context(), schoolID, gradeID)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	// Decorate each subject with its effective mark-component scheme (its
	// own custom one if it has one, else the grade-template default) so the
	// marks-entry UI never has to guess or fetch it separately per subject.
	out := make([]subjectResponse, len(items))
	for i, s := range items {
		components, err := m.repo.GetSubjectComponents(r.Context(), s.ID)
		if err != nil {
			httpx.WriteServiceError(w, err)
			return
		}
		out[i] = subjectResponse{Subject: s, MarkComponents: components}
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{"items": out})
}

// subjectResponse decorates a Subject with its effective mark-component
// scheme, without persisting either field differently on the subject itself.
type subjectResponse struct {
	domain.Subject
	MarkComponents []MarkComponent `json:"mark_components,omitempty"`
}

func (m *Module) UpdateSubject(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	var in domain.Subject
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid json")
		return
	}
	in.ID = id
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		httpx.Error(w, http.StatusBadRequest, "subject name is required")
		return
	}
	if !in.IsCoScholastic {
		if in.MaxMarks <= 0 {
			httpx.Error(w, http.StatusBadRequest, "max marks must be more than 0")
			return
		}
		if in.PassingMarks < 0 || in.PassingMarks > in.MaxMarks {
			httpx.Error(w, http.StatusBadRequest, "passing marks must be between 0 and the max marks")
			return
		}
	}
	if err := m.repo.UpdateSubject(r.Context(), &in); err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, in)
}

func (m *Module) DeleteSubject(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := m.repo.DeleteSubject(r.Context(), id); err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.NoContent(w)
}

// ---- Per-subject mark components ("sections") ----

func (m *Module) ListSubjectComponents(w http.ResponseWriter, r *http.Request) {
	subjectID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	components, err := m.repo.GetSubjectComponents(r.Context(), subjectID)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{"items": components})
}

func (m *Module) AddSubjectComponent(w http.ResponseWriter, r *http.Request) {
	subjectID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	var in struct {
		Label    string `json:"label"`
		MaxMarks int    `json:"max_marks"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid json")
		return
	}
	in.Label = strings.TrimSpace(in.Label)
	key := slugifyComponentKey(in.Label)
	if key == "" || in.MaxMarks <= 0 {
		httpx.Error(w, http.StatusBadRequest, "label and a positive max_marks are required")
		return
	}
	var createdBy uuid.UUID
	if claims := httpx.ClaimsFromContext(r.Context()); claims != nil {
		createdBy, _ = uuid.Parse(claims.Sub)
	}
	components, err := m.repo.AddSubjectComponent(r.Context(), subjectID, key, in.Label, in.MaxMarks, createdBy)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, map[string]interface{}{"items": components})
}

// UpdateSubjectComponent renames a subject's mark field and/or changes its
// max marks (e.g. Written 60 -> 50). Max marks can't change once marks are
// recorded under the field.
func (m *Module) UpdateSubjectComponent(w http.ResponseWriter, r *http.Request) {
	subjectID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	key := chi.URLParam(r, "key")
	var in struct {
		Label    string `json:"label"`
		MaxMarks int    `json:"max_marks"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid json")
		return
	}
	in.Label = strings.TrimSpace(in.Label)
	if in.MaxMarks < 0 || (in.Label == "" && in.MaxMarks == 0) {
		httpx.Error(w, http.StatusBadRequest, "send a label and/or a positive max_marks")
		return
	}
	var createdBy uuid.UUID
	if claims := httpx.ClaimsFromContext(r.Context()); claims != nil {
		createdBy, _ = uuid.Parse(claims.Sub)
	}
	components, err := m.repo.UpdateSubjectComponent(r.Context(), subjectID, key, in.Label, in.MaxMarks, createdBy)
	if err != nil {
		if errors.Is(err, apperr.ErrConflict) {
			httpx.Error(w, http.StatusConflict, "marks are already recorded for this field, so its max marks can't change -- you can still rename it")
			return
		}
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{"items": components})
}

func (m *Module) DeleteSubjectComponent(w http.ResponseWriter, r *http.Request) {
	subjectID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	key := chi.URLParam(r, "key")
	var createdBy uuid.UUID
	if claims := httpx.ClaimsFromContext(r.Context()); claims != nil {
		createdBy, _ = uuid.Parse(claims.Sub)
	}
	if err := m.repo.DeleteSubjectComponent(r.Context(), subjectID, key, createdBy); err != nil {
		if errors.Is(err, apperr.ErrConflict) {
			httpx.Error(w, http.StatusConflict, "this section already has marks recorded against it and can't be removed")
			return
		}
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.NoContent(w)
}

// ---- Exam handlers ----

func (m *Module) CreateExam(w http.ResponseWriter, r *http.Request) {
	// exam_date arrives as a bare "YYYY-MM-DD" from an <input type="date">, not
	// full RFC3339 -- decoding straight into domain.Exam's *time.Time field
	// fails the whole request the moment a date is picked (Go's time.Time
	// JSON unmarshaling requires a time+timezone component). Decode the date
	// as a string here and parse it explicitly instead.
	var in struct {
		SchoolID       uuid.UUID `json:"school_id"`
		AcademicYearID uuid.UUID `json:"academic_year_id"`
		GradeLevelID   uuid.UUID `json:"grade_level_id"`
		Name           string    `json:"name"`
		ExamDate       string    `json:"exam_date"`
		WeightPercent  int       `json:"weight_percent"`
		IsPublished    bool      `json:"is_published"`
		FeeLockEnabled bool      `json:"fee_lock_enabled"`
		FeeLockMinDue  int       `json:"fee_lock_min_due"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid json")
		return
	}
	e := domain.Exam{
		SchoolID:       in.SchoolID,
		AcademicYearID: in.AcademicYearID,
		GradeLevelID:   in.GradeLevelID,
		Name:           in.Name,
		WeightPercent:  in.WeightPercent,
		IsPublished:    in.IsPublished,
		FeeLockEnabled: in.FeeLockEnabled,
		FeeLockMinDue:  in.FeeLockMinDue,
	}
	if e.FeeLockMinDue < 0 {
		httpx.Error(w, http.StatusBadRequest, "fee lock amount can't be negative")
		return
	}
	if in.ExamDate != "" {
		d, err := time.Parse("2006-01-02", in.ExamDate)
		if err != nil {
			httpx.Error(w, http.StatusBadRequest, "invalid exam_date, expected YYYY-MM-DD")
			return
		}
		e.ExamDate = &d
	}
	if e.WeightPercent <= 0 {
		e.WeightPercent = 100
	}
	if err := m.repo.CreateExam(r.Context(), &e); err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, e)
}

func (m *Module) ListExams(w http.ResponseWriter, r *http.Request) {
	schoolID, _ := uuid.Parse(r.URL.Query().Get("school_id"))
	yearID, _ := uuid.Parse(r.URL.Query().Get("academic_year_id"))
	gradeID, _ := uuid.Parse(r.URL.Query().Get("grade_level_id"))
	if schoolID == uuid.Nil || yearID == uuid.Nil || gradeID == uuid.Nil {
		httpx.Error(w, http.StatusBadRequest, "school_id, academic_year_id, grade_level_id required")
		return
	}
	if ok, err := m.teacherCanAccessGradeInYear(r.Context(), httpx.ClaimsFromContext(r.Context()), yearID, gradeID); err != nil {
		httpx.WriteServiceError(w, err)
		return
	} else if !ok {
		httpx.Error(w, http.StatusForbidden, "access denied: not your class")
		return
	}
	items, err := m.repo.ListExams(r.Context(), schoolID, yearID, gradeID)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	// Decorate with the grade's mark-component scheme so the marks-entry UI
	// never has to hardcode a second copy of Written/Note Book/.../Theory.
	template, err := m.repo.GetGradeReportTemplate(r.Context(), gradeID)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	out := make([]examResponse, len(items))
	for i, e := range items {
		out[i] = examResponse{Exam: e, MarkComponents: MarkComponentsForTemplate(template)}
		if template != nil {
			out[i].ReportCardTemplate = *template
		}
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{"items": out})
}

// examResponse decorates an Exam with its grade's mark-component scheme for
// the marks-entry UI, without persisting either field on the exam itself.
type examResponse struct {
	domain.Exam
	ReportCardTemplate string          `json:"report_card_template,omitempty"`
	MarkComponents     []MarkComponent `json:"mark_components,omitempty"`
}

// ListExamFormats returns each subject's marks distribution for one exam.
func (m *Module) ListExamFormats(w http.ResponseWriter, r *http.Request) {
	examID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	gradeID, yearID, err := m.repo.ExamGradeAndYear(r.Context(), examID)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	if ok, err := m.teacherCanAccessGradeInYear(r.Context(), httpx.ClaimsFromContext(r.Context()), yearID, gradeID); err != nil {
		httpx.WriteServiceError(w, err)
		return
	} else if !ok {
		httpx.Error(w, http.StatusForbidden, "access denied: not your class")
		return
	}
	items, err := m.repo.ListExamFormats(r.Context(), examID)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{"items": items})
}

// SetExamSubjectFormat sets one subject's fields for one exam, e.g.
// {"components":[{"label":"Written","max_marks":20}]}; an empty list means one
// plain mark. Send a field's existing key to rename it without losing marks.
func (m *Module) SetExamSubjectFormat(w http.ResponseWriter, r *http.Request) {
	examID, err1 := uuid.Parse(chi.URLParam(r, "id"))
	subjectID, err2 := uuid.Parse(chi.URLParam(r, "subjectId"))
	if err1 != nil || err2 != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	var in struct {
		Components []MarkComponent `json:"components"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid json")
		return
	}
	components, err := NormalizeComponents(in.Components)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	var createdBy uuid.UUID
	if claims := httpx.ClaimsFromContext(r.Context()); claims != nil {
		createdBy, _ = uuid.Parse(claims.Sub)
	}
	if err := m.repo.SetExamSubjectFormat(r.Context(), examID, subjectID, components, createdBy); err != nil {
		if errors.Is(err, ErrFormatHasMarks) {
			httpx.Error(w, http.StatusConflict, strings.TrimPrefix(err.Error(), ErrFormatHasMarks.Error()+": "))
			return
		}
		httpx.WriteServiceError(w, err)
		return
	}
	m.writeExamFormats(w, r, examID)
}

// ResetExamSubjectFormat makes a subject use its own fields again in one exam.
func (m *Module) ResetExamSubjectFormat(w http.ResponseWriter, r *http.Request) {
	examID, err1 := uuid.Parse(chi.URLParam(r, "id"))
	subjectID, err2 := uuid.Parse(chi.URLParam(r, "subjectId"))
	if err1 != nil || err2 != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := m.repo.ResetExamSubjectFormat(r.Context(), examID, subjectID); err != nil {
		if errors.Is(err, ErrFormatHasMarks) {
			httpx.Error(w, http.StatusConflict, strings.TrimPrefix(err.Error(), ErrFormatHasMarks.Error()+": "))
			return
		}
		httpx.WriteServiceError(w, err)
		return
	}
	m.writeExamFormats(w, r, examID)
}

func (m *Module) writeExamFormats(w http.ResponseWriter, r *http.Request, examID uuid.UUID) {
	items, err := m.repo.ListExamFormats(r.Context(), examID)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{"items": items})
}

// UpdateExam edits an exam's name, date (YYYY-MM-DD, "" clears it) and weight.
func (m *Module) UpdateExam(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	var in struct {
		Name          string `json:"name"`
		ExamDate      string `json:"exam_date"`
		WeightPercent int    `json:"weight_percent"`
		// Optional: left out, the exam keeps its current fee lock.
		FeeLockEnabled *bool `json:"fee_lock_enabled"`
		FeeLockMinDue  *int  `json:"fee_lock_min_due"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid json")
		return
	}
	e, err := m.repo.GetExamByID(r.Context(), id)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	e.Name = strings.TrimSpace(in.Name)
	if e.Name == "" {
		httpx.Error(w, http.StatusBadRequest, "exam name is required")
		return
	}
	if in.WeightPercent < 1 || in.WeightPercent > 100 {
		httpx.Error(w, http.StatusBadRequest, "weight must be between 1 and 100")
		return
	}
	e.WeightPercent = in.WeightPercent
	if in.FeeLockEnabled != nil {
		e.FeeLockEnabled = *in.FeeLockEnabled
	}
	if in.FeeLockMinDue != nil {
		if *in.FeeLockMinDue < 0 {
			httpx.Error(w, http.StatusBadRequest, "fee lock amount can't be negative")
			return
		}
		e.FeeLockMinDue = *in.FeeLockMinDue
	}
	e.ExamDate = nil
	if in.ExamDate != "" {
		d, err := time.Parse("2006-01-02", in.ExamDate)
		if err != nil {
			httpx.Error(w, http.StatusBadRequest, "invalid exam_date, expected YYYY-MM-DD")
			return
		}
		e.ExamDate = &d
	}
	if err := m.repo.UpdateExam(r.Context(), e); err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, e)
}

// DeleteExam removes an exam that is unpublished and has no marks entered.
func (m *Module) DeleteExam(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := m.repo.DeleteExam(r.Context(), id); err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.NoContent(w)
}

// GetResultSheet returns one exam's marks for its whole class.
func (m *Module) GetResultSheet(w http.ResponseWriter, r *http.Request) {
	examID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	// Exams are fetched by id, so check the caller's school here: a school
	// admin, registrar or teacher only sees their own school's sheets.
	claims := httpx.ClaimsFromContext(r.Context())
	if claims != nil && claims.Role != "super_admin" && claims.SchoolID != "" {
		e, err := m.repo.GetExamByID(r.Context(), examID)
		if err != nil {
			httpx.WriteServiceError(w, err)
			return
		}
		if e.SchoolID.String() != claims.SchoolID {
			httpx.Error(w, http.StatusForbidden, "access denied: school mismatch")
			return
		}
	}
	// A class teacher sees only the students of their own section(s).
	teacherID, restricted, err := m.teacherIDIfRestricted(r.Context(), claims)
	if err != nil {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var mine map[uuid.UUID]bool
	if restricted {
		e, err := m.repo.GetExamByID(r.Context(), examID)
		if err != nil {
			httpx.WriteServiceError(w, err)
			return
		}
		if mine, err = m.repo.TeacherStudentIDs(r.Context(), teacherID, e.AcademicYearID, e.GradeLevelID); err != nil {
			httpx.WriteServiceError(w, err)
			return
		}
		if len(mine) == 0 {
			httpx.Error(w, http.StatusForbidden, "access denied: you are not the class teacher of this class")
			return
		}
	}
	sheet, err := m.repo.GetResultSheet(r.Context(), examID)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	if restricted {
		sheet.OnlyStudents(mine)
	}
	httpx.JSON(w, http.StatusOK, sheet)
}

func (m *Module) PublishExam(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	var body struct {
		Publish bool `json:"publish"`
		// Optional: set the fee lock in the same step as publishing.
		FeeLockEnabled *bool `json:"fee_lock_enabled"`
		FeeLockMinDue  *int  `json:"fee_lock_min_due"`
	}
	json.NewDecoder(r.Body).Decode(&body)
	if body.FeeLockEnabled != nil || body.FeeLockMinDue != nil {
		e, err := m.repo.GetExamByID(r.Context(), id)
		if err != nil {
			httpx.WriteServiceError(w, err)
			return
		}
		if body.FeeLockEnabled != nil {
			e.FeeLockEnabled = *body.FeeLockEnabled
		}
		if body.FeeLockMinDue != nil {
			if *body.FeeLockMinDue < 0 {
				httpx.Error(w, http.StatusBadRequest, "fee lock amount can't be negative")
				return
			}
			e.FeeLockMinDue = *body.FeeLockMinDue
		}
		if err := m.repo.UpdateExam(r.Context(), e); err != nil {
			httpx.WriteServiceError(w, err)
			return
		}
	}
	if err := m.repo.PublishExam(r.Context(), id, body.Publish); err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]bool{"published": body.Publish})
}

// ---- Mark handlers ----

func (m *Module) UpsertMark(w http.ResponseWriter, r *http.Request) {
	var mark domain.ExamMark
	if err := json.NewDecoder(r.Body).Decode(&mark); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid json")
		return
	}
	if mark.MaxMarks <= 0 {
		mark.MaxMarks = 100
	}
	if ok, err := m.teacherCanAccessMarks(r.Context(), httpx.ClaimsFromContext(r.Context()), []domain.ExamMark{mark}); err != nil {
		httpx.WriteServiceError(w, err)
		return
	} else if !ok {
		httpx.Error(w, http.StatusForbidden, "access denied: not your class")
		return
	}
	if err := m.saveOneMark(r.Context(), &mark); err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, mark)
}

// saveOneMark routes through the component-aware path when the mark carries
// a component breakdown, otherwise saves the plain total as before.
//
// The scheme used here MUST be the subject's own effective one
// (GetSubjectComponents: its stored custom components, or the grade
// template's if it has none of its own) -- not the raw grade template
// directly. Using the template directly was a real bug: a subject with its
// own custom field added beyond the template (e.g. an extra "Copy" column)
// would have that field's value silently dropped on save, since
// UpsertMarkWithComponents only persists whatever's in the components list
// it's given.
// gradeMarkError checks a mark that carries a letter grade: the letter must be
// one of GradeLetters and the subject must be grading-only. ok is false for
// an ordinary marks entry (no letter), which is saved as before.
func (m *Module) gradeMarkError(ctx context.Context, mark *domain.ExamMark, coScholastic map[uuid.UUID]bool) (ok bool, err error) {
	mark.GradeLetter = strings.ToUpper(strings.TrimSpace(mark.GradeLetter))
	if mark.GradeLetter == "" {
		return false, nil
	}
	valid := false
	for _, g := range GradeLetters {
		valid = valid || g == mark.GradeLetter
	}
	if !valid {
		return false, fmt.Errorf("%w: grade must be one of %s", apperr.ErrInvalidInput, strings.Join(GradeLetters, ", "))
	}
	cs, seen := coScholastic[mark.SubjectID]
	if !seen {
		if cs, err = m.repo.SubjectIsCoScholastic(ctx, mark.SubjectID); err != nil {
			return false, err
		}
		coScholastic[mark.SubjectID] = cs
	}
	if !cs {
		return false, fmt.Errorf("%w: only grading-only subjects take a letter grade -- enter marks for this subject", apperr.ErrInvalidInput)
	}
	return true, nil
}

func (m *Module) saveOneMark(ctx context.Context, mark *domain.ExamMark) error {
	if isGrade, err := m.gradeMarkError(ctx, mark, map[uuid.UUID]bool{}); err != nil {
		return err
	} else if isGrade {
		return m.repo.UpsertGradeMark(ctx, mark)
	}
	if len(mark.Components) == 0 {
		return m.repo.UpsertMark(ctx, mark)
	}
	components, err := m.repo.EffectiveComponents(ctx, mark.ExamID, mark.SubjectID)
	if err != nil {
		return err
	}
	if len(components) == 0 {
		// Grade has no template and subject has no custom components -- fall
		// back to treating the submitted total as a plain mark.
		return m.repo.UpsertMark(ctx, mark)
	}
	return m.repo.UpsertMarkWithComponents(ctx, mark, components)
}

func (m *Module) BulkUpsertMarks(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Marks []domain.ExamMark `json:"marks"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid json")
		return
	}
	if ok, err := m.teacherCanAccessMarks(r.Context(), httpx.ClaimsFromContext(r.Context()), body.Marks); err != nil {
		httpx.WriteServiceError(w, err)
		return
	} else if !ok {
		httpx.Error(w, http.StatusForbidden, "access denied: not your class")
		return
	}
	saved, err := m.bulkUpsert(r.Context(), body.Marks)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{"saved": saved})
}

func (m *Module) bulkUpsert(ctx context.Context, marks []domain.ExamMark) (int, error) {
	// A bulk save is typically one exam across a whole class -- many rows,
	// same subjects -- so cache each (exam, subject) scheme instead of
	// re-querying it per row. Each subject can have its own fields, and its
	// own format per exam (see EffectiveComponents).
	type schemeKey struct{ examID, subjectID uuid.UUID }
	schemes := map[schemeKey][]MarkComponent{}
	var saved int
	// Check every letter grade before saving anything, so a bad one rejects
	// the whole save instead of leaving it half done.
	coScholastic := map[uuid.UUID]bool{}
	isGrade := make([]bool, len(marks))
	for i := range marks {
		ok, err := m.gradeMarkError(ctx, &marks[i], coScholastic)
		if err != nil {
			return 0, err
		}
		isGrade[i] = ok
	}
	for i := range marks {
		mark := &marks[i]
		if isGrade[i] {
			if err := m.repo.UpsertGradeMark(ctx, mark); err != nil {
				return saved, err
			}
			saved++
			continue
		}
		if mark.MaxMarks <= 0 {
			mark.MaxMarks = 100
		}
		if len(mark.Components) == 0 {
			if err := m.repo.UpsertMark(ctx, mark); err != nil {
				return saved, err
			}
			saved++
			continue
		}
		sk := schemeKey{mark.ExamID, mark.SubjectID}
		components, ok := schemes[sk]
		if !ok {
			var err error
			components, err = m.repo.EffectiveComponents(ctx, mark.ExamID, mark.SubjectID)
			if err != nil {
				return saved, err
			}
			schemes[sk] = components
		}
		if len(components) == 0 {
			if err := m.repo.UpsertMark(ctx, mark); err != nil {
				return saved, err
			}
		} else if err := m.repo.UpsertMarkWithComponents(ctx, mark, components); err != nil {
			return saved, err
		}
		saved++
	}
	return saved, nil
}

// ---- Marksheet handlers ----

// ListStudentExamMarks returns what is already saved for a student in an
// exam, so the marks entry form opens with it filled in.
func (m *Module) ListStudentExamMarks(w http.ResponseWriter, r *http.Request) {
	examID, err := uuid.Parse(r.URL.Query().Get("exam_id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "exam_id required")
		return
	}
	studentID, err := uuid.Parse(r.URL.Query().Get("student_id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "student_id required")
		return
	}
	if ok, err := m.teacherCanAccessExamStudent(r.Context(), httpx.ClaimsFromContext(r.Context()), examID, studentID); err != nil {
		httpx.WriteServiceError(w, err)
		return
	} else if !ok {
		httpx.Error(w, http.StatusForbidden, "access denied: not your class")
		return
	}
	items, err := m.repo.StudentExamMarks(r.Context(), examID, studentID)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{"items": items})
}

func (m *Module) GetMarksheet(w http.ResponseWriter, r *http.Request) {
	examID, err := uuid.Parse(r.URL.Query().Get("exam_id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "exam_id required")
		return
	}
	studentID, err := uuid.Parse(r.URL.Query().Get("student_id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "student_id required")
		return
	}
	if ok, err := m.teacherCanAccessExamStudent(r.Context(), httpx.ClaimsFromContext(r.Context()), examID, studentID); err != nil {
		httpx.WriteServiceError(w, err)
		return
	} else if !ok {
		httpx.Error(w, http.StatusForbidden, "access denied: not your class")
		return
	}
	if !m.parentMayViewExam(w, r, examID, studentID) {
		return
	}
	ms, err := m.repo.GetStudentMarksheet(r.Context(), examID, studentID)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, ms)
}

func (m *Module) DownloadMarksheet(w http.ResponseWriter, r *http.Request) {
	examID, err := uuid.Parse(r.URL.Query().Get("exam_id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "exam_id required")
		return
	}
	studentID, err := uuid.Parse(r.URL.Query().Get("student_id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "student_id required")
		return
	}
	if ok, err := m.teacherCanAccessExamStudent(r.Context(), httpx.ClaimsFromContext(r.Context()), examID, studentID); err != nil {
		httpx.WriteServiceError(w, err)
		return
	} else if !ok {
		httpx.Error(w, http.StatusForbidden, "access denied: not your class")
		return
	}
	if !m.parentMayViewExam(w, r, examID, studentID) {
		return
	}
	ms, err := m.repo.GetStudentMarksheet(r.Context(), examID, studentID)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	if len(ms.Rows) == 0 {
		httpx.Error(w, http.StatusNotFound, "no marks found for this student")
		return
	}
	pdfBytes, err := generateMarksheetPDF(*ms)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "pdf generation failed")
		return
	}
	m.archive.Save(r.Context(), studentID, archive.KindMarksheet, "Marksheet: "+ms.ExamName, pdfBytes)
	filename := fmt.Sprintf("marksheet_%s_%s.pdf", ms.StudentCode, examID.String()[:8])
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.Header().Set("Content-Length", strconv.Itoa(len(pdfBytes)))
	w.WriteHeader(http.StatusOK)
	w.Write(pdfBytes)
}

// SetTotalOverride lets a super_admin manually correct the total shown on a
// student's marksheet (e.g. a moderation adjustment) without re-entering
// every subject's marks. The new total must fall within [0, total_max] as
// already computed from the subjects on record.
func (m *Module) SetTotalOverride(w http.ResponseWriter, r *http.Request) {
	var in struct {
		ExamID        string  `json:"exam_id"`
		StudentID     string  `json:"student_id"`
		TotalObtained float64 `json:"total_obtained"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid json")
		return
	}
	examID, err := uuid.Parse(in.ExamID)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid exam_id")
		return
	}
	studentID, err := uuid.Parse(in.StudentID)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid student_id")
		return
	}

	current, err := m.repo.GetStudentMarksheet(r.Context(), examID, studentID)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	// TotalMax is always the auto-computed max regardless of any prior
	// override -- only TotalObtained is ever replaced.
	maxTotal := current.TotalMax
	if in.TotalObtained < 0 || (maxTotal > 0 && in.TotalObtained > float64(maxTotal)) {
		httpx.Error(w, http.StatusBadRequest, fmt.Sprintf("total_obtained must be between 0 and %d", maxTotal))
		return
	}

	var overriddenBy uuid.UUID
	if claims := httpx.ClaimsFromContext(r.Context()); claims != nil {
		overriddenBy, _ = uuid.Parse(claims.Sub)
	}
	if err := m.repo.SetTotalOverride(r.Context(), examID, studentID, in.TotalObtained, overriddenBy); err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	ms, err := m.repo.GetStudentMarksheet(r.Context(), examID, studentID)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, ms)
}

// DeleteTotalOverride reverts a marksheet to its auto-calculated total.
func (m *Module) DeleteTotalOverride(w http.ResponseWriter, r *http.Request) {
	examID, err := uuid.Parse(r.URL.Query().Get("exam_id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "exam_id required")
		return
	}
	studentID, err := uuid.Parse(r.URL.Query().Get("student_id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "student_id required")
		return
	}
	if err := m.repo.DeleteTotalOverride(r.Context(), examID, studentID); err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	ms, err := m.repo.GetStudentMarksheet(r.Context(), examID, studentID)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, ms)
}

// ---- Report card handlers (combined multi-exam, class-wise templates) ----

func (m *Module) getReportCardForRequest(r *http.Request) (*ReportCard, error) {
	studentID, err := uuid.Parse(r.URL.Query().Get("student_id"))
	if err != nil {
		return nil, apperr.ErrInvalidInput
	}
	yearID, err := uuid.Parse(r.URL.Query().Get("academic_year_id"))
	if err != nil {
		return nil, apperr.ErrInvalidInput
	}
	if ok, err := m.teacherOrParentCanAccessStudentInYear(r.Context(), httpx.ClaimsFromContext(r.Context()), studentID, yearID); err != nil {
		return nil, err
	} else if !ok {
		return nil, apperr.ErrForbidden
	}
	claims := httpx.ClaimsFromContext(r.Context())
	isParent := claims != nil && claims.Role == "parent"
	if isParent {
		if lock, err := m.reportCardFeeLock(r.Context(), studentID, yearID); err != nil {
			return nil, err
		} else if lock != nil {
			return nil, lock
		}
	}
	rc, err := m.repo.GetReportCard(r.Context(), studentID, yearID, isParent)
	if err != nil {
		return nil, err
	}
	m.attachParentNames(r.Context(), rc)
	return rc, nil
}

// attachParentNames fills Father/Mother name from the existing guardian
// module rather than duplicating that lookup here. Best-effort: a lookup
// failure leaves the names blank instead of failing the whole report card.
func (m *Module) attachParentNames(ctx context.Context, rc *ReportCard) {
	guardians, err := m.wards.ListByStudent(ctx, rc.StudentID)
	if err != nil {
		return
	}
	for _, g := range guardians {
		name := strings.TrimSpace(strings.TrimSpace(g.FirstName) + " " + strings.TrimSpace(g.LastName))
		switch strings.ToLower(strings.TrimSpace(g.Relation)) {
		case "father":
			rc.FatherName = name
		case "mother":
			rc.MotherName = name
		}
	}
}

func (m *Module) GetReportCard(w http.ResponseWriter, r *http.Request) {
	rc, err := m.getReportCardForRequest(r)
	if err != nil {
		if errors.Is(err, apperr.ErrInvalidInput) {
			httpx.Error(w, http.StatusBadRequest, "student_id and academic_year_id required")
			return
		}
		var lock *feeLockError
		if errors.As(err, &lock) {
			writeFeeLocked(w, lock.due)
			return
		}
		if errors.Is(err, apperr.ErrForbidden) {
			httpx.Error(w, http.StatusForbidden, "access denied: not your class")
			return
		}
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, rc)
}

func (m *Module) DownloadReportCard(w http.ResponseWriter, r *http.Request) {
	rc, err := m.getReportCardForRequest(r)
	if err != nil {
		if errors.Is(err, apperr.ErrInvalidInput) {
			httpx.Error(w, http.StatusBadRequest, "student_id and academic_year_id required")
			return
		}
		var lock *feeLockError
		if errors.As(err, &lock) {
			writeFeeLocked(w, lock.due)
			return
		}
		if errors.Is(err, apperr.ErrForbidden) {
			httpx.Error(w, http.StatusForbidden, "access denied: not your class")
			return
		}
		httpx.WriteServiceError(w, err)
		return
	}
	theme := themeByName(r.URL.Query().Get("design"))
	var pdfBytes []byte
	switch rc.Template {
	case "kg":
		pdfBytes, err = generateKGReportCardPDF(*rc, theme)
	case "primary":
		pdfBytes, err = generatePrimaryReportCardPDF(*rc, theme)
	case "middle":
		pdfBytes, err = generateMiddleReportCardPDF(*rc, theme)
	default:
		httpx.Error(w, http.StatusBadRequest, "no report-card template for this grade")
		return
	}
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "pdf generation failed")
		return
	}
	m.archive.Save(r.Context(), rc.StudentID, archive.KindReportCard, "Report card", pdfBytes)
	filename := fmt.Sprintf("report_card_%s_%s.pdf", rc.StudentCode, rc.AcademicYearID.String()[:8])
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.Header().Set("Content-Length", strconv.Itoa(len(pdfBytes)))
	w.WriteHeader(http.StatusOK)
	w.Write(pdfBytes)
}

func (m *Module) UpsertReportCardDetails(w http.ResponseWriter, r *http.Request) {
	var in domain.ReportCardDetails
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid json")
		return
	}
	if in.SchoolID == uuid.Nil || in.AcademicYearID == uuid.Nil || in.GradeLevelID == uuid.Nil || in.StudentID == uuid.Nil {
		httpx.Error(w, http.StatusBadRequest, "school_id, academic_year_id, grade_level_id, student_id required")
		return
	}
	if ok, err := m.teacherOrParentCanAccessStudentInYear(r.Context(), httpx.ClaimsFromContext(r.Context()), in.StudentID, in.AcademicYearID); err != nil {
		httpx.WriteServiceError(w, err)
		return
	} else if !ok {
		httpx.Error(w, http.StatusForbidden, "access denied: not your class")
		return
	}
	if err := m.repo.UpsertReportCardDetails(r.Context(), &in); err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, in)
}

// ListDisciplineCriteria exposes the "middle" template's fixed 9 criteria so
// the frontend never hardcodes a second copy.
func (m *Module) ListDisciplineCriteria(w http.ResponseWriter, r *http.Request) {
	httpx.JSON(w, http.StatusOK, map[string]interface{}{"items": disciplineCriteria})
}

func (m *Module) UpsertDisciplineGrades(w http.ResponseWriter, r *http.Request) {
	var body struct {
		SchoolID       uuid.UUID         `json:"school_id"`
		AcademicYearID uuid.UUID         `json:"academic_year_id"`
		StudentID      uuid.UUID         `json:"student_id"`
		Grades         map[string]string `json:"grades"` // criterion_key -> grade
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid json")
		return
	}
	if body.SchoolID == uuid.Nil || body.AcademicYearID == uuid.Nil || body.StudentID == uuid.Nil {
		httpx.Error(w, http.StatusBadRequest, "school_id, academic_year_id, student_id required")
		return
	}
	if ok, err := m.teacherOrParentCanAccessStudentInYear(r.Context(), httpx.ClaimsFromContext(r.Context()), body.StudentID, body.AcademicYearID); err != nil {
		httpx.WriteServiceError(w, err)
		return
	} else if !ok {
		httpx.Error(w, http.StatusForbidden, "access denied: not your class")
		return
	}
	for key, grade := range body.Grades {
		g := domain.DisciplineGrade{SchoolID: body.SchoolID, AcademicYearID: body.AcademicYearID, StudentID: body.StudentID, CriterionKey: key, Grade: grade}
		if err := m.repo.UpsertDisciplineGrade(r.Context(), &g); err != nil {
			httpx.WriteServiceError(w, err)
			return
		}
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "saved"})
}

// ---- Class-teacher scoping helpers ----
//
// A teacher only sees/enters results for the grade(s)/student(s) of the class
// section(s) where they are the homeroom teacher. Every other role is unaffected.

// teacherIDIfRestricted returns the requester's user id and restricted=true
// whenever the requester is a teacher -- scoped strictly to whichever class
// section(s), if any, have them set as homeroom_teacher_id. A teacher with no
// homeroom class assigned yet sees/can touch nothing in Results, rather than
// falling back to unrestricted access: now that admins have a real place to
// assign class teachers (Settings -> Grades & Sections), "not assigned yet"
// should mean "no class", not "every class".
func (m *Module) teacherIDIfRestricted(ctx context.Context, claims *httpx.Claims) (teacherID uuid.UUID, restricted bool, err error) {
	if claims == nil || claims.Role != "teacher" {
		return uuid.Nil, false, nil
	}
	teacherID, err = uuid.Parse(claims.Sub)
	if err != nil {
		return uuid.Nil, false, err
	}
	return teacherID, true, nil
}

func (m *Module) teacherCanAccessGrade(ctx context.Context, claims *httpx.Claims, gradeID uuid.UUID) (bool, error) {
	teacherID, restricted, err := m.teacherIDIfRestricted(ctx, claims)
	if err != nil || !restricted {
		return err == nil, err
	}
	return m.repo.TeacherOwnsGrade(ctx, teacherID, gradeID)
}

func (m *Module) teacherCanAccessGradeInYear(ctx context.Context, claims *httpx.Claims, yearID, gradeID uuid.UUID) (bool, error) {
	teacherID, restricted, err := m.teacherIDIfRestricted(ctx, claims)
	if err != nil || !restricted {
		return err == nil, err
	}
	return m.repo.TeacherOwnsGradeInYear(ctx, teacherID, yearID, gradeID)
}

// teacherOrParentCanAccessStudentInYear is teacherCanAccessExamStudent's
// counterpart for report-card endpoints, which key on (student, academic
// year) directly rather than resolving the year from an exam.
func (m *Module) teacherOrParentCanAccessStudentInYear(ctx context.Context, claims *httpx.Claims, studentID, yearID uuid.UUID) (bool, error) {
	if claims != nil && claims.Role == "parent" {
		return m.parentCanAccessStudent(ctx, claims, studentID)
	}
	teacherID, restricted, err := m.teacherIDIfRestricted(ctx, claims)
	if err != nil || !restricted {
		return err == nil, err
	}
	return m.repo.TeacherOwnsStudent(ctx, teacherID, yearID, studentID)
}

func (m *Module) teacherCanAccessExamStudent(ctx context.Context, claims *httpx.Claims, examID, studentID uuid.UUID) (bool, error) {
	if claims != nil && claims.Role == "parent" {
		return m.parentCanAccessStudent(ctx, claims, studentID)
	}
	teacherID, restricted, err := m.teacherIDIfRestricted(ctx, claims)
	if err != nil || !restricted {
		return err == nil, err
	}
	yearID, err := m.repo.ExamAcademicYear(ctx, examID)
	if err != nil {
		return false, err
	}
	return m.repo.TeacherOwnsStudent(ctx, teacherID, yearID, studentID)
}

// WardExams returns the published exams for a parent's ward's current class.
func (m *Module) WardExams(w http.ResponseWriter, r *http.Request) {
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
	claims := httpx.ClaimsFromContext(r.Context())
	if ok, err := m.parentCanAccessStudent(r.Context(), claims, studentID); err != nil {
		httpx.WriteServiceError(w, err)
		return
	} else if !ok {
		httpx.Error(w, http.StatusForbidden, "access denied: not your ward")
		return
	}
	items, err := m.repo.WardExams(r.Context(), studentID, yearID)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	// Mark the exams whose results the fee lock hides from this parent, so
	// the app can say so instead of opening them.
	type wardExam struct {
		domain.Exam
		FeeLocked bool `json:"fee_locked"`
		FeeDue    int  `json:"fee_due,omitempty"`
	}
	out := make([]wardExam, 0, len(items))
	due, dueKnown := 0, false
	for _, e := range items {
		we := wardExam{Exam: e}
		if e.FeeLockEnabled {
			if !dueKnown {
				if due, err = m.repo.FeeDue(r.Context(), studentID, yearID); err != nil {
					httpx.WriteServiceError(w, err)
					return
				}
				dueKnown = true
			}
			if due > e.FeeLockMinDue {
				we.FeeLocked, we.FeeDue = true, due
			}
		}
		out = append(out, we)
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{"items": out})
}

// parentCanAccessStudent restricts a parent to marksheets for their own ward(s) only.
func (m *Module) parentCanAccessStudent(ctx context.Context, claims *httpx.Claims, studentID uuid.UUID) (bool, error) {
	if claims == nil {
		return false, nil
	}
	userID, err := uuid.Parse(claims.Sub)
	if err != nil {
		return false, nil
	}
	ids, err := m.wards.WardStudentIDs(ctx, userID)
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

// teacherCanAccessMarks checks every mark's (exam, student) pair against the teacher's
// homeroom class, caching each exam's academic year to avoid repeat lookups in a bulk save.
func (m *Module) teacherCanAccessMarks(ctx context.Context, claims *httpx.Claims, marks []domain.ExamMark) (bool, error) {
	teacherID, restricted, err := m.teacherIDIfRestricted(ctx, claims)
	if err != nil || !restricted {
		return err == nil, err
	}
	examYears := map[uuid.UUID]uuid.UUID{}
	for _, mk := range marks {
		yearID, ok := examYears[mk.ExamID]
		if !ok {
			yearID, err = m.repo.ExamAcademicYear(ctx, mk.ExamID)
			if err != nil {
				return false, err
			}
			examYears[mk.ExamID] = yearID
		}
		owns, err := m.repo.TeacherOwnsStudent(ctx, teacherID, yearID, mk.StudentID)
		if err != nil {
			return false, err
		}
		if !owns {
			return false, nil
		}
	}
	return true, nil
}

// ---- Results fee lock ----

// feeLockError is returned when a parent's fee due hides results from them.
type feeLockError struct{ due int }

func (e *feeLockError) Error() string { return fmt.Sprintf("results locked: %d due", e.due) }

// writeFeeLocked tells a parent their results are locked, in words the app
// can show as-is (older app versions just show the error text).
func writeFeeLocked(w http.ResponseWriter, due int) {
	httpx.JSON(w, http.StatusForbidden, map[string]interface{}{
		"error":      fmt.Sprintf("Results are locked: ₹%s fee is due. Please pay at the school office to see the results.", rupees(due)),
		"fee_locked": true,
		"fee_due":    due,
	})
}

// rupees formats an amount the Indian way, e.g. 125000 -> "1,25,000".
func rupees(n int) string {
	s := strconv.Itoa(n)
	if len(s) <= 3 {
		return s
	}
	head, tail := s[:len(s)-3], s[len(s)-3:]
	var parts []string
	for len(head) > 2 {
		parts = append([]string{head[len(head)-2:]}, parts...)
		head = head[:len(head)-2]
	}
	if head != "" {
		parts = append([]string{head}, parts...)
	}
	return strings.Join(parts, ",") + "," + tail
}

// parentMayViewExam lets staff through untouched. For a parent it refuses an
// exam that isn't published yet, and one whose fee lock applies to them,
// writing the response itself; it returns false when it did.
func (m *Module) parentMayViewExam(w http.ResponseWriter, r *http.Request, examID, studentID uuid.UUID) bool {
	claims := httpx.ClaimsFromContext(r.Context())
	if claims == nil || claims.Role != "parent" {
		return true
	}
	e, err := m.repo.GetExamByID(r.Context(), examID)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return false
	}
	if !e.IsPublished {
		httpx.Error(w, http.StatusNotFound, "these results are not published yet")
		return false
	}
	if !e.FeeLockEnabled {
		return true
	}
	due, err := m.repo.FeeDue(r.Context(), studentID, e.AcademicYearID)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return false
	}
	if due > e.FeeLockMinDue {
		writeFeeLocked(w, due)
		return false
	}
	return true
}

// reportCardFeeLock returns a *feeLockError when any published, fee-locked
// exam of the student's class hides the year's report card from a parent.
func (m *Module) reportCardFeeLock(ctx context.Context, studentID, yearID uuid.UUID) (error, error) {
	minDue, any, err := m.repo.StrictestFeeLock(ctx, studentID, yearID)
	if err != nil || !any {
		return nil, err
	}
	due, err := m.repo.FeeDue(ctx, studentID, yearID)
	if err != nil {
		return nil, err
	}
	if due > minDue {
		return &feeLockError{due: due}, nil
	}
	return nil, nil
}
