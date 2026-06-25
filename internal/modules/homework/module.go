package homework

import (
	"encoding/json"
	"net/http"
	"time"

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

func (m *Module) Name() string { return "homework" }

func (m *Module) Mount(r chi.Router) {
	r.Route("/homework", func(r chi.Router) {
		r.Get("/", m.ListAssignments)
		r.Post("/", m.CreateAssignment)
		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", m.GetAssignment)
			r.Delete("/", m.DeleteAssignment)
			r.Route("/submissions", func(r chi.Router) {
				r.Get("/", m.ListSubmissions)
				r.Post("/", m.UpsertSubmission)
			})
		})
	})
	r.Get("/homework-tracker", m.StudentTracker)
}

// ---- Assignment handlers ----

func (m *Module) CreateAssignment(w http.ResponseWriter, r *http.Request) {
	var in struct {
		SchoolID       string `json:"school_id"`
		AcademicYearID string `json:"academic_year_id"`
		GradeLevelID   string `json:"grade_level_id"`
		ClassSectionID string `json:"class_section_id"`
		SubjectID      string `json:"subject_id"`
		Title          string `json:"title"`
		Description    string `json:"description"`
		AssignedBy     string `json:"assigned_by"`
		AssignedDate   string `json:"assigned_date"`
		DueDate        string `json:"due_date"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid json")
		return
	}
	schoolID, _ := uuid.Parse(in.SchoolID)
	yearID, _ := uuid.Parse(in.AcademicYearID)
	gradeID, _ := uuid.Parse(in.GradeLevelID)
	if schoolID == uuid.Nil || yearID == uuid.Nil || gradeID == uuid.Nil || in.Title == "" {
		httpx.Error(w, http.StatusBadRequest, "school_id, academic_year_id, grade_level_id, title required")
		return
	}

	a := &domain.HomeworkAssignment{
		SchoolID:       schoolID,
		AcademicYearID: yearID,
		GradeLevelID:   gradeID,
		Title:          in.Title,
		Description:    in.Description,
	}

	if cs, err := uuid.Parse(in.ClassSectionID); err == nil && cs != uuid.Nil {
		a.ClassSectionID = &cs
	}
	if sub, err := uuid.Parse(in.SubjectID); err == nil && sub != uuid.Nil {
		a.SubjectID = &sub
	}
	if by, err := uuid.Parse(in.AssignedBy); err == nil && by != uuid.Nil {
		a.AssignedBy = &by
	}
	if t, err := time.Parse("2006-01-02", in.AssignedDate); err == nil {
		a.AssignedDate = t
	} else {
		a.AssignedDate = time.Now()
	}
	if t, err := time.Parse("2006-01-02", in.DueDate); err == nil {
		a.DueDate = t
	} else {
		httpx.Error(w, http.StatusBadRequest, "due_date required (YYYY-MM-DD)")
		return
	}

	if err := m.repo.CreateAssignment(r.Context(), a); err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, a)
}

func (m *Module) GetAssignment(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	a, err := m.repo.GetAssignment(r.Context(), id)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, a)
}

func (m *Module) ListAssignments(w http.ResponseWriter, r *http.Request) {
	schoolID, _ := uuid.Parse(r.URL.Query().Get("school_id"))
	yearID, _ := uuid.Parse(r.URL.Query().Get("academic_year_id"))
	gradeID, _ := uuid.Parse(r.URL.Query().Get("grade_level_id"))
	if schoolID == uuid.Nil || yearID == uuid.Nil || gradeID == uuid.Nil {
		httpx.Error(w, http.StatusBadRequest, "school_id, academic_year_id, grade_level_id required")
		return
	}
	items, err := m.repo.ListAssignments(r.Context(), schoolID, yearID, gradeID)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{"items": items})
}

func (m *Module) DeleteAssignment(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := m.repo.DeleteAssignment(r.Context(), id); err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.NoContent(w)
}

// ---- Submission handlers ----

func (m *Module) UpsertSubmission(w http.ResponseWriter, r *http.Request) {
	assignmentID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid assignment id")
		return
	}
	var in struct {
		StudentID     string `json:"student_id"`
		SubmittedDate string `json:"submitted_date"`
		Status        string `json:"status"` // pending / submitted / late / missing
		Remarks       string `json:"remarks"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid json")
		return
	}
	studentID, _ := uuid.Parse(in.StudentID)
	if studentID == uuid.Nil {
		httpx.Error(w, http.StatusBadRequest, "student_id required")
		return
	}
	if in.Status == "" {
		in.Status = "submitted"
	}

	s := &domain.HomeworkSubmission{
		AssignmentID: assignmentID,
		StudentID:    studentID,
		Status:       in.Status,
		Remarks:      in.Remarks,
	}
	if t, err := time.Parse("2006-01-02", in.SubmittedDate); err == nil {
		s.SubmittedDate = &t
	} else {
		now := time.Now()
		s.SubmittedDate = &now
	}

	if err := m.repo.UpsertSubmission(r.Context(), s); err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, s)
}

func (m *Module) ListSubmissions(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid assignment id")
		return
	}
	items, err := m.repo.ListSubmissions(r.Context(), id)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{"items": items})
}

// StudentTracker returns all homework submissions for a student in a year
func (m *Module) StudentTracker(w http.ResponseWriter, r *http.Request) {
	studentID, _ := uuid.Parse(r.URL.Query().Get("student_id"))
	yearID, _ := uuid.Parse(r.URL.Query().Get("academic_year_id"))
	if studentID == uuid.Nil || yearID == uuid.Nil {
		httpx.Error(w, http.StatusBadRequest, "student_id and academic_year_id required")
		return
	}
	items, err := m.repo.GetStudentSubmissions(r.Context(), studentID, yearID)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{"items": items})
}
