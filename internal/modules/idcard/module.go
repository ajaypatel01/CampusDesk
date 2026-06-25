package idcard

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/ajaypatel01/CampusDesk/internal/platform/httpx"
	"github.com/ajaypatel01/CampusDesk/internal/platform/storage"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v4/pgxpool"
)

type Module struct {
	svc *Service
}

func New(pool *pgxpool.Pool, s *storage.Client) *Module {
	repo := NewRepository(pool)
	svc := NewService(repo, s)
	return &Module{svc: svc}
}

func (m *Module) Name() string { return "idcard" }

func (m *Module) Mount(r chi.Router) {
	r.Post("/id-cards/students", m.StudentCards)
	r.Post("/id-cards/teachers", m.TeacherCards)
}

func (m *Module) StudentCards(w http.ResponseWriter, r *http.Request) {
	var body struct {
		StudentIDs     []string `json:"student_ids"`
		AcademicYearID string   `json:"academic_year_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid json body")
		return
	}
	yearID, err := uuid.Parse(body.AcademicYearID)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "academic_year_id required")
		return
	}
	var ids []uuid.UUID
	for _, s := range body.StudentIDs {
		if id, err := uuid.Parse(s); err == nil {
			ids = append(ids, id)
		}
	}
	pdfBytes, filename, err := m.svc.GenerateStudentCards(r.Context(), ids, yearID)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	servePDF(w, pdfBytes, filename)
}

func (m *Module) TeacherCards(w http.ResponseWriter, r *http.Request) {
	var body struct {
		UserIDs []string `json:"user_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid json body")
		return
	}
	var ids []uuid.UUID
	for _, s := range body.UserIDs {
		if id, err := uuid.Parse(s); err == nil {
			ids = append(ids, id)
		}
	}
	pdfBytes, filename, err := m.svc.GenerateTeacherCards(r.Context(), ids)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	servePDF(w, pdfBytes, filename)
}

func servePDF(w http.ResponseWriter, data []byte, filename string) {
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}
