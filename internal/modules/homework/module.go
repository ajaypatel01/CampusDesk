package homework

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/ajaypatel01/CampusDesk/internal/domain"
	"github.com/ajaypatel01/CampusDesk/internal/modules/guardian"
	"github.com/ajaypatel01/CampusDesk/internal/platform/httpx"
	"github.com/ajaypatel01/CampusDesk/internal/platform/storage"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v4/pgxpool"
)

type Module struct {
	repo    *Repository
	wards   *guardian.Repository
	storage *storage.Client
}

func New(pool *pgxpool.Pool, s *storage.Client) *Module {
	return &Module{repo: NewRepository(pool), wards: guardian.NewRepository(pool), storage: s}
}

func (m *Module) Name() string { return "homework" }

const feature = "homework"

func (m *Module) Mount(r chi.Router) {
	view := httpx.RequireFeature(feature, "view")
	write := httpx.RequireFeature(feature, "write")
	r.Route("/homework", func(r chi.Router) {
		r.With(httpx.BlockRoles("parent"), view).Get("/", m.ListAssignments)
		r.With(httpx.BlockRoles("parent"), write).Post("/", m.CreateAssignment)
		r.Route("/{id}", func(r chi.Router) {
			r.With(view).Get("/", m.GetAssignment)
			r.With(httpx.BlockRoles("parent"), write).Delete("/", m.DeleteAssignment)
			r.Route("/submissions", func(r chi.Router) {
				r.With(view).Get("/", m.ListSubmissions)
				r.With(httpx.BlockRoles("parent"), write).Post("/", m.UpsertSubmission)
			})
		})
	})
	r.With(view).Get("/homework-tracker", m.StudentTracker)
	r.With(view).Get("/ward-homework", m.WardHomework)
}

// isWard reports whether the current request's claims belong to a parent whose
// portal access includes studentID. Non-parent roles always return true (unaffected).
func (m *Module) isWard(r *http.Request, studentID uuid.UUID) (bool, error) {
	claims := httpx.ClaimsFromContext(r.Context())
	if claims == nil || claims.Role != "parent" {
		return true, nil
	}
	userID, err := uuid.Parse(claims.Sub)
	if err != nil {
		return false, nil
	}
	ids, err := m.wards.WardStudentIDs(r.Context(), userID)
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

// WardHomework returns a parent's ward's class homework (grade/section-wide
// assignments) together with that ward's own submission status for each.
func (m *Module) WardHomework(w http.ResponseWriter, r *http.Request) {
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
	if ok, err := m.isWard(r, studentID); err != nil {
		httpx.WriteServiceError(w, err)
		return
	} else if !ok {
		httpx.Error(w, http.StatusForbidden, "access denied: not your ward")
		return
	}
	items, err := m.repo.GetWardHomework(r.Context(), studentID, yearID)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	for i := range items {
		m.withLink(&items[i].HomeworkAssignment)
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{"items": items})
}

// ---- Assignment handlers ----

// CreateAssignment takes JSON, or multipart form fields plus an optional
// "file" (one image or PDF). Teachers may only set homework for a section
// where they are the class or vice class teacher; class, year and school
// then come from that section. Admins/registrars: their own school.
func (m *Module) CreateAssignment(w http.ResponseWriter, r *http.Request) {
	claims := httpx.ClaimsFromContext(r.Context())
	in := map[string]string{}
	var fileData []byte
	var fileName string
	if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/") {
		r.Body = http.MaxBytesReader(w, r.Body, maxAttachment+(1<<20))
		if err := r.ParseMultipartForm(4 << 20); err != nil {
			httpx.Error(w, http.StatusBadRequest, "file too large (at most 10 MB)")
			return
		}
		for k, v := range r.MultipartForm.Value {
			if len(v) > 0 {
				in[k] = v[0]
			}
		}
		if f, fh, err := r.FormFile("file"); err == nil {
			fileData, err = io.ReadAll(f)
			f.Close()
			if err != nil {
				httpx.Error(w, http.StatusBadRequest, "could not read the file")
				return
			}
			fileName = fh.Filename
		}
	} else if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid json")
		return
	}

	a := &domain.HomeworkAssignment{Title: strings.TrimSpace(in["title"]), Description: strings.TrimSpace(in["description"])}
	if a.Title == "" {
		httpx.Error(w, http.StatusBadRequest, "title required")
		return
	}
	sectionID, _ := uuid.Parse(in["class_section_id"])
	if claims.Role == "teacher" {
		me, _ := uuid.Parse(claims.Sub)
		secs, err := m.repo.TeacherSections(r.Context(), me)
		if err != nil {
			httpx.WriteServiceError(w, err)
			return
		}
		var mine *TeacherSection
		for i := range secs {
			if secs[i].ID == sectionID {
				mine = &secs[i]
			}
		}
		if mine == nil {
			httpx.Error(w, http.StatusForbidden, "you can only set homework for your own class")
			return
		}
		a.SchoolID, a.AcademicYearID, a.GradeLevelID, a.ClassSectionID = mine.SchoolID, mine.AcademicYearID, mine.GradeLevelID, &mine.ID
	} else {
		a.SchoolID, _ = uuid.Parse(in["school_id"])
		a.AcademicYearID, _ = uuid.Parse(in["academic_year_id"])
		a.GradeLevelID, _ = uuid.Parse(in["grade_level_id"])
		if a.SchoolID == uuid.Nil || a.AcademicYearID == uuid.Nil || a.GradeLevelID == uuid.Nil {
			httpx.Error(w, http.StatusBadRequest, "school_id, academic_year_id, grade_level_id, title required")
			return
		}
		if claims.Role != "super_admin" && a.SchoolID.String() != claims.SchoolID {
			httpx.Error(w, http.StatusForbidden, "access denied: not your school")
			return
		}
		if sectionID != uuid.Nil {
			a.ClassSectionID = &sectionID
		}
	}
	if sub, err := uuid.Parse(in["subject_id"]); err == nil && sub != uuid.Nil {
		a.SubjectID = &sub
	}
	if by, err := uuid.Parse(claims.Sub); err == nil {
		a.AssignedBy = &by
	}
	if t, err := time.Parse("2006-01-02", in["assigned_date"]); err == nil {
		a.AssignedDate = t
	} else {
		a.AssignedDate = time.Now()
	}
	t, err := time.Parse("2006-01-02", in["due_date"])
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "due_date required (YYYY-MM-DD)")
		return
	}
	a.DueDate = t

	if len(fileData) > 0 {
		if !m.storage.Enabled() {
			httpx.Error(w, http.StatusServiceUnavailable, "file storage is not set up yet")
			return
		}
		ctype, ext, err := checkAttachment(fileData)
		if err != nil {
			httpx.Error(w, http.StatusBadRequest, err.Error())
			return
		}
		key := fmt.Sprintf("homework/%s/%s%s", a.SchoolID, uuid.New(), ext)
		if err := m.storage.Upload(key, ctype, fileData); err != nil {
			httpx.Error(w, http.StatusBadGateway, "could not store the file, please try again")
			return
		}
		a.AttachmentKey, a.AttachmentType, a.AttachmentSize = key, ctype, len(fileData)
		a.AttachmentName = filepath.Base(strings.ReplaceAll(fileName, "\\", "/"))
	}
	if err := m.repo.CreateAssignment(r.Context(), a); err != nil {
		if a.AttachmentKey != "" {
			_ = m.storage.Delete(a.AttachmentKey)
		}
		httpx.WriteServiceError(w, err)
		return
	}
	m.withLink(a)
	httpx.JSON(w, http.StatusCreated, a)
}

const maxAttachment = 10 << 20

// checkAttachment accepts a PDF or an image, judged by content.
func checkAttachment(data []byte) (string, string, error) {
	if len(data) > maxAttachment {
		return "", "", errors.New("file larger than 10 MB")
	}
	if bytes.HasPrefix(data, []byte("%PDF")) {
		return "application/pdf", ".pdf", nil
	}
	switch ct := http.DetectContentType(data); ct {
	case "image/jpeg":
		return ct, ".jpg", nil
	case "image/png":
		return ct, ".png", nil
	case "image/webp":
		return ct, ".webp", nil
	case "image/gif":
		return ct, ".gif", nil
	}
	return "", "", errors.New("attach a PDF or an image (JPG, PNG, WEBP)")
}

// withLink fills in a short-lived link to the attachment.
func (m *Module) withLink(a *domain.HomeworkAssignment) {
	if a.AttachmentKey != "" && m.storage.Enabled() {
		a.AttachmentURL, _ = m.storage.PresignedDownloadURL(a.AttachmentKey, a.AttachmentName, 2*time.Hour)
	}
}

// teacherScope returns the caller's sections when they're a teacher.
func (m *Module) teacherScope(r *http.Request) (secs []TeacherSection, isTeacher bool, err error) {
	claims := httpx.ClaimsFromContext(r.Context())
	if claims == nil || claims.Role != "teacher" {
		return nil, false, nil
	}
	me, _ := uuid.Parse(claims.Sub)
	secs, err = m.repo.TeacherSections(r.Context(), me)
	return secs, true, err
}

// mayManage: teachers only for their own sections (or homework they set);
// others within their school.
func (m *Module) mayManage(r *http.Request, a *domain.HomeworkAssignment) (bool, error) {
	claims := httpx.ClaimsFromContext(r.Context())
	secs, isTeacher, err := m.teacherScope(r)
	if err != nil {
		return false, err
	}
	if !isTeacher {
		return claims.Role == "super_admin" || a.SchoolID.String() == claims.SchoolID, nil
	}
	if a.AssignedBy != nil && a.AssignedBy.String() == claims.Sub {
		return true, nil
	}
	for _, s := range secs {
		if a.ClassSectionID != nil && *a.ClassSectionID == s.ID {
			return true, nil
		}
	}
	return false, nil
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
	m.withLink(a)
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
	// Teachers see only homework for their own section(s), or grade-wide
	// homework of a grade they teach.
	secs, isTeacher, err := m.teacherScope(r)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	out := make([]domain.HomeworkAssignment, 0, len(items))
	for _, a := range items {
		if isTeacher && !teacherSees(secs, a) {
			continue
		}
		m.withLink(&a)
		out = append(out, a)
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{"items": out})
}

func teacherSees(secs []TeacherSection, a domain.HomeworkAssignment) bool {
	for _, s := range secs {
		if a.ClassSectionID != nil && *a.ClassSectionID == s.ID {
			return true
		}
		if a.ClassSectionID == nil && a.GradeLevelID == s.GradeLevelID && a.AcademicYearID == s.AcademicYearID {
			return true
		}
	}
	return false
}

func (m *Module) DeleteAssignment(w http.ResponseWriter, r *http.Request) {
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
	if ok, err := m.mayManage(r, a); err != nil {
		httpx.WriteServiceError(w, err)
		return
	} else if !ok {
		httpx.Error(w, http.StatusForbidden, "you can only delete homework for your own class")
		return
	}
	if err := m.repo.DeleteAssignment(r.Context(), id); err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	if a.AttachmentKey != "" && m.storage.Enabled() {
		_ = m.storage.Delete(a.AttachmentKey)
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
	if hw, err := m.repo.GetAssignment(r.Context(), assignmentID); err != nil {
		httpx.WriteServiceError(w, err)
		return
	} else if ok, err := m.mayManage(r, hw); err != nil {
		httpx.WriteServiceError(w, err)
		return
	} else if !ok {
		httpx.Error(w, http.StatusForbidden, "you can only mark homework for your own class")
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
	if ok, err := m.isWard(r, studentID); err != nil {
		httpx.WriteServiceError(w, err)
		return
	} else if !ok {
		httpx.Error(w, http.StatusForbidden, "access denied: not your ward")
		return
	}
	items, err := m.repo.GetStudentSubmissions(r.Context(), studentID, yearID)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{"items": items})
}
