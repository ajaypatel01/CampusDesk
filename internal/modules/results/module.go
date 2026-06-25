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
		r.Post("/", m.CreateSubject)
		r.Put("/{id}", m.UpdateSubject)
		r.Delete("/{id}", m.DeleteSubject)
	})
	r.Route("/exams", func(r chi.Router) {
		r.Get("/", m.ListExams)
		r.Post("/", m.CreateExam)
		r.Post("/{id}/publish", m.PublishExam)
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

