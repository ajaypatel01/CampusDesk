package results

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
	repo  *Repository
	wards *guardian.Repository
}

func New(pool *pgxpool.Pool) *Module {
	return &Module{repo: NewRepository(pool), wards: guardian.NewRepository(pool)}
}

func (m *Module) Name() string { return "results" }

func (m *Module) Mount(r chi.Router) {
	r.Route("/subjects", func(r chi.Router) {
		r.With(httpx.BlockRoles("parent")).Get("/", m.ListSubjects)
		// Subject/exam setup is admin work; class teachers only view and enter marks.
		// Parents never manage results data, only view their own ward's marksheet below.
		r.With(httpx.BlockRoles("teacher", "parent")).Post("/", m.CreateSubject)
		r.With(httpx.BlockRoles("teacher", "parent")).Put("/{id}", m.UpdateSubject)
		r.With(httpx.BlockRoles("teacher", "parent")).Delete("/{id}", m.DeleteSubject)
	})
	r.Route("/exams", func(r chi.Router) {
		r.With(httpx.BlockRoles("parent")).Get("/", m.ListExams)
		r.With(httpx.BlockRoles("teacher", "parent")).Post("/", m.CreateExam)
		r.With(httpx.BlockRoles("teacher", "parent")).Post("/{id}/publish", m.PublishExam)
	})
	r.Route("/exam-marks", func(r chi.Router) {
		r.With(httpx.BlockRoles("parent")).Post("/", m.UpsertMark)
		r.With(httpx.BlockRoles("parent")).Post("/bulk", m.BulkUpsertMarks)
	})
	r.Get("/ward-exams", m.WardExams)
	r.Route("/marksheets", func(r chi.Router) {
		r.Get("/", m.GetMarksheet)
		r.Get("/pdf", m.DownloadMarksheet)
	})
	r.Route("/report-cards", func(r chi.Router) {
		r.Get("/", m.GetReportCard)
		r.Get("/pdf", m.DownloadReportCard)
		// Attendance/remark/promoted-to and discipline grades are filled in
		// by the class teacher, same access as entering marks -- open to
		// teacher/admin, never parent.
		r.With(httpx.BlockRoles("parent")).Put("/details", m.UpsertReportCardDetails)
		r.With(httpx.BlockRoles("parent")).Put("/discipline-grades", m.UpsertDisciplineGrades)
	})
	r.Get("/discipline-criteria", m.ListDisciplineCriteria)
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
	httpx.JSON(w, http.StatusOK, map[string]interface{}{"items": items})
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

func (m *Module) PublishExam(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	var body struct {
		Publish bool `json:"publish"`
	}
	json.NewDecoder(r.Body).Decode(&body)
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
// a component breakdown (i.e. its exam's grade has a report-card template
// and the entry form sent one), otherwise saves the plain total as before.
func (m *Module) saveOneMark(ctx context.Context, mark *domain.ExamMark) error {
	if len(mark.Components) == 0 {
		return m.repo.UpsertMark(ctx, mark)
	}
	template, err := m.repo.GetExamReportTemplate(ctx, mark.ExamID)
	if err != nil {
		return err
	}
	components := MarkComponentsForTemplate(template)
	if len(components) == 0 {
		// Grade has no template (or none matching a known scheme) -- fall
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
	// A bulk save is typically one exam across a whole class, so cache each
	// exam's component scheme instead of re-querying it per row.
	schemes := map[uuid.UUID][]MarkComponent{}
	var saved int
	for i := range marks {
		mark := &marks[i]
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
		components, ok := schemes[mark.ExamID]
		if !ok {
			template, err := m.repo.GetExamReportTemplate(ctx, mark.ExamID)
			if err != nil {
				return saved, err
			}
			components = MarkComponentsForTemplate(template)
			schemes[mark.ExamID] = components
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
	filename := fmt.Sprintf("marksheet_%s_%s.pdf", ms.StudentCode, examID.String()[:8])
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.Header().Set("Content-Length", strconv.Itoa(len(pdfBytes)))
	w.WriteHeader(http.StatusOK)
	w.Write(pdfBytes)
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
	rc, err := m.repo.GetReportCard(r.Context(), studentID, yearID)
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
		if errors.Is(err, apperr.ErrForbidden) {
			httpx.Error(w, http.StatusForbidden, "access denied: not your class")
			return
		}
		httpx.WriteServiceError(w, err)
		return
	}
	var pdfBytes []byte
	switch rc.Template {
	case "kg":
		pdfBytes, err = generateKGReportCardPDF(*rc)
	case "primary":
		pdfBytes, err = generatePrimaryReportCardPDF(*rc)
	case "middle":
		pdfBytes, err = generateMiddleReportCardPDF(*rc)
	default:
		httpx.Error(w, http.StatusBadRequest, "no report-card template for this grade")
		return
	}
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "pdf generation failed")
		return
	}
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

// teacherIDIfRestricted returns the requester's user id and restricted=true only when
// the requester is a teacher who is homeroom teacher of at least one class section
// somewhere. A teacher with no homeroom class assigned yet (nothing set up) falls back
// to unrestricted access instead of being locked out of Results entirely.
func (m *Module) teacherIDIfRestricted(ctx context.Context, claims *httpx.Claims) (teacherID uuid.UUID, restricted bool, err error) {
	if claims == nil || claims.Role != "teacher" {
		return uuid.Nil, false, nil
	}
	teacherID, err = uuid.Parse(claims.Sub)
	if err != nil {
		return uuid.Nil, false, err
	}
	hasHomeroom, err := m.repo.TeacherHasAnyHomeroom(ctx, teacherID)
	if err != nil {
		return uuid.Nil, false, err
	}
	return teacherID, hasHomeroom, nil
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
	httpx.JSON(w, http.StatusOK, map[string]interface{}{"items": items})
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
