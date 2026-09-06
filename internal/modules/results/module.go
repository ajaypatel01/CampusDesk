package results

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/ajaypatel01/CampusDesk/internal/domain"
	"github.com/ajaypatel01/CampusDesk/internal/platform/httpx"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v4/pgxpool"
)

type Module struct {
	repo *Repository
}

func New(pool *pgxpool.Pool) *Module {
	return &Module{repo: NewRepository(pool)}
}

func (m *Module) Name() string { return "results" }

func (m *Module) Mount(r chi.Router) {
	r.Route("/subjects", func(r chi.Router) {
		r.Get("/", m.ListSubjects)
		// Subject/exam setup is admin work; class teachers only view and enter marks.
		r.With(httpx.BlockRoles("teacher")).Post("/", m.CreateSubject)
		r.With(httpx.BlockRoles("teacher")).Put("/{id}", m.UpdateSubject)
		r.With(httpx.BlockRoles("teacher")).Delete("/{id}", m.DeleteSubject)
	})
	r.Route("/exams", func(r chi.Router) {
		r.Get("/", m.ListExams)
		r.With(httpx.BlockRoles("teacher")).Post("/", m.CreateExam)
		r.With(httpx.BlockRoles("teacher")).Post("/{id}/publish", m.PublishExam)
	})
	r.Route("/exam-marks", func(r chi.Router) {
		r.Post("/", m.UpsertMark)
		r.Post("/bulk", m.BulkUpsertMarks)
	})
	r.Route("/marksheets", func(r chi.Router) {
		r.Get("/", m.GetMarksheet)
		r.Get("/pdf", m.DownloadMarksheet)
	})
}

// ---- Subject handlers ----

func (m *Module) CreateSubject(w http.ResponseWriter, r *http.Request) {
	var in struct {
		SchoolID     string `json:"school_id"`
		GradeLevelID string `json:"grade_level_id"`
		Name         string `json:"name"`
		Code         string `json:"code"`
		MaxMarks     int    `json:"max_marks"`
		PassingMarks int    `json:"passing_marks"`
		SortOrder    int    `json:"sort_order"`
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
	var e domain.Exam
	if err := json.NewDecoder(r.Body).Decode(&e); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid json")
		return
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
	httpx.JSON(w, http.StatusOK, map[string]interface{}{"items": items})
}

func (m *Module) PublishExam(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	var body struct{ Publish bool `json:"publish"` }
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
	if err := m.repo.UpsertMark(r.Context(), &mark); err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, mark)
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
	var saved int
	for i := range marks {
		if marks[i].MaxMarks <= 0 {
			marks[i].MaxMarks = 100
		}
		if err := m.repo.UpsertMark(ctx, &marks[i]); err != nil {
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

func (m *Module) teacherCanAccessExamStudent(ctx context.Context, claims *httpx.Claims, examID, studentID uuid.UUID) (bool, error) {
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

