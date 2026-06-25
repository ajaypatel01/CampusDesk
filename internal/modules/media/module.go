package media

import (
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/ajaypatel01/CampusDesk/internal/platform/httpx"
	"github.com/ajaypatel01/CampusDesk/internal/platform/storage"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v4/pgxpool"
)

type Module struct {
	pool    *pgxpool.Pool
	storage *storage.Client
}

func New(pool *pgxpool.Pool, s *storage.Client) *Module {
	return &Module{pool: pool, storage: s}
}

func (m *Module) Name() string { return "media" }

func (m *Module) Mount(r chi.Router) {
	r.Post("/media/students/{id}/photo", m.UploadStudentPhoto)
	r.Post("/media/users/{id}/photo", m.UploadUserPhoto)
}

const maxPhotoSize = 5 << 20 // 5 MB

func (m *Module) UploadStudentPhoto(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid student id")
		return
	}
	key, url, err := m.uploadPhoto(r, "students", id.String())
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if _, err := m.pool.Exec(r.Context(),
		`UPDATE students SET photo_url=$2, updated_at=NOW() WHERE id=$1`, id, key); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "db update failed")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"photo_url": url, "key": key})
}

func (m *Module) UploadUserPhoto(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid user id")
		return
	}
	key, url, err := m.uploadPhoto(r, "users", id.String())
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if _, err := m.pool.Exec(r.Context(),
		`UPDATE users SET photo_url=$2, updated_at=NOW() WHERE id=$1`, id, key); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "db update failed")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"photo_url": url, "key": key})
}

func (m *Module) uploadPhoto(r *http.Request, entityType, entityID string) (key, url string, err error) {
	if !m.storage.Enabled() {
		return "", "", fmt.Errorf("storage not configured")
	}
	r.Body = http.MaxBytesReader(nil, r.Body, maxPhotoSize)
	if err := r.ParseMultipartForm(maxPhotoSize); err != nil {
		return "", "", fmt.Errorf("parse form: max 5MB allowed")
	}
	file, header, err := r.FormFile("photo")
	if err != nil {
		return "", "", fmt.Errorf("photo field required")
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" {
		return "", "", fmt.Errorf("only JPG and PNG allowed")
	}
	contentType := "image/jpeg"
	if ext == ".png" {
		contentType = "image/png"
	}

	data, err := io.ReadAll(file)
	if err != nil {
		return "", "", fmt.Errorf("read file: %w", err)
	}

	key = fmt.Sprintf("photos/%s/%s%s", entityType, entityID, ext)
	if err := m.storage.Upload(key, contentType, data); err != nil {
		return "", "", fmt.Errorf("upload failed: %w", err)
	}
	return key, key, nil
}
