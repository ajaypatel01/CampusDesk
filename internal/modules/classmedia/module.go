// Package classmedia is the class gallery (event photos) and class
// documents: files uploaded for one class section and kept in S3.
//
// Who can do what, per section:
//   - owner (super_admin): view and manage any section;
//   - school_admin, registrar: view and manage their school's sections;
//   - teacher: view and manage sections where they are the class or vice
//     class teacher;
//   - parent: view their children's sections.
//
// Uploaders may also delete what they uploaded.
package classmedia

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	apperr "github.com/ajaypatel01/CampusDesk/internal/platform/errors"
	"github.com/ajaypatel01/CampusDesk/internal/platform/httpx"
	"github.com/ajaypatel01/CampusDesk/internal/platform/storage"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v4/pgxpool"
)

const (
	KindPhoto    = "photo"
	KindDocument = "document"

	maxPhotoBytes    = 15 << 20
	maxDocumentBytes = 20 << 20
	maxRequestBytes  = 25 << 20
	maxFilesPerPost  = 10
	urlTTL           = 2 * time.Hour
)

type Module struct {
	repo    *Repository
	storage *storage.Client
}

func New(pool *pgxpool.Pool, s *storage.Client) *Module {
	return &Module{repo: &Repository{pool: pool}, storage: s}
}

func (m *Module) Name() string { return "classmedia" }

func (m *Module) Mount(r chi.Router) {
	r.Get("/class-media/sections", m.ListSections)
	r.Get("/class-media", m.List)
	r.Post("/class-media", m.Upload)
	r.Delete("/class-media/{id}", m.Delete)
	r.With(httpx.RequireRole("super_admin", "school_admin", "registrar")).Get("/issued-documents", m.ListIssued)
	// A student's own papers (passbook, Aadhaar card, certificates): staff who manage records only.
	staff := httpx.RequireRole("super_admin", "school_admin", "registrar")
	r.With(staff).Get("/students/{id}/documents", m.ListStudentDocs)
	r.With(staff).Post("/students/{id}/documents", m.UploadStudentDoc)
	r.With(staff).Delete("/student-documents/{id}", m.DeleteStudentDoc)
}

// ---- Access ----

type access struct {
	view, manage bool
}

func (m *Module) sectionAccess(r *http.Request, sectionID uuid.UUID) (access, *Section, error) {
	claims := httpx.ClaimsFromContext(r.Context())
	if claims == nil {
		return access{}, nil, apperr.ErrUnauthorized
	}
	sec, err := m.repo.Section(r.Context(), sectionID)
	if err != nil {
		return access{}, nil, err
	}
	userID, _ := uuid.Parse(claims.Sub)
	switch claims.Role {
	case "super_admin":
		return access{true, true}, sec, nil
	case "school_admin", "registrar":
		same := sec.SchoolID.String() == claims.SchoolID
		return access{same, same}, sec, nil
	case "teacher":
		mine, err := m.repo.TeachesSection(r.Context(), userID, sectionID)
		return access{mine, mine}, sec, err
	case "parent":
		mine, err := m.repo.ParentOfSection(r.Context(), userID, sectionID)
		return access{mine, false}, sec, err
	}
	return access{}, sec, nil
}

// ---- Handlers ----

// ListSections returns the sections the caller can open, current year first.
func (m *Module) ListSections(w http.ResponseWriter, r *http.Request) {
	claims := httpx.ClaimsFromContext(r.Context())
	if claims == nil {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	userID, _ := uuid.Parse(claims.Sub)
	var schoolID uuid.UUID
	if claims.Role == "super_admin" {
		id, err := uuid.Parse(r.URL.Query().Get("school_id"))
		if err != nil {
			httpx.Error(w, http.StatusBadRequest, "school_id required")
			return
		}
		schoolID = id
	} else if id, err := uuid.Parse(claims.SchoolID); err == nil {
		schoolID = id
	}
	secs, err := m.repo.Sections(r.Context(), claims.Role, userID, schoolID)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{"items": secs, "storage_ready": m.storage.Enabled()})
}

// List returns one section's photos or documents, newest first, with
// short-lived links to the files.
func (m *Module) List(w http.ResponseWriter, r *http.Request) {
	sectionID, err := uuid.Parse(r.URL.Query().Get("section_id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "section_id required")
		return
	}
	kind := r.URL.Query().Get("kind")
	if kind != KindPhoto && kind != KindDocument {
		httpx.Error(w, http.StatusBadRequest, "kind must be photo or document")
		return
	}
	acc, _, err := m.sectionAccess(r, sectionID)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	if !acc.view {
		httpx.Error(w, http.StatusForbidden, "access denied: not your class")
		return
	}
	items, err := m.repo.List(r.Context(), sectionID, kind)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	me := httpx.ClaimsFromContext(r.Context()).Sub
	for i := range items {
		it := &items[i]
		it.CanDelete = acc.manage || (it.UploadedBy != nil && it.UploadedBy.String() == me)
		if m.storage.Enabled() {
			it.URL, _ = m.storage.PresignedDownloadURL(it.s3Key, it.FileName, urlTTL)
			if it.thumbKey != "" {
				it.ThumbURL, _ = m.storage.PresignedURL(it.thumbKey, urlTTL)
			}
		}
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{"items": items, "can_upload": acc.manage})
}

// Upload stores one or more files (multipart "files") for a section.
// Apps send one file per request so each stays small.
func (m *Module) Upload(w http.ResponseWriter, r *http.Request) {
	if !m.storage.Enabled() {
		httpx.Error(w, http.StatusServiceUnavailable, "file storage is not set up yet")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBytes)
	if err := r.ParseMultipartForm(8 << 20); err != nil {
		httpx.Error(w, http.StatusBadRequest, "upload too large (at most 15 MB per photo, 20 MB per document)")
		return
	}
	sectionID, err := uuid.Parse(r.FormValue("section_id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "section_id required")
		return
	}
	kind := r.FormValue("kind")
	if kind != KindPhoto && kind != KindDocument {
		httpx.Error(w, http.StatusBadRequest, "kind must be photo or document")
		return
	}
	acc, sec, err := m.sectionAccess(r, sectionID)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	if !acc.manage {
		httpx.Error(w, http.StatusForbidden, "access denied: you can only upload to your own class")
		return
	}
	event := strings.TrimSpace(r.FormValue("event"))
	if len(event) > 120 {
		event = event[:120]
	}
	var eventDate *time.Time
	if s := r.FormValue("event_date"); s != "" {
		d, err := time.Parse("2006-01-02", s)
		if err != nil {
			httpx.Error(w, http.StatusBadRequest, "event_date must be YYYY-MM-DD")
			return
		}
		eventDate = &d
	}
	files := r.MultipartForm.File["files"]
	if len(files) == 0 {
		httpx.Error(w, http.StatusBadRequest, "no files")
		return
	}
	if len(files) > maxFilesPerPost {
		httpx.Error(w, http.StatusBadRequest, fmt.Sprintf("at most %d files at a time", maxFilesPerPost))
		return
	}
	uploaderID, _ := uuid.Parse(httpx.ClaimsFromContext(r.Context()).Sub)

	var saved []Item
	for _, fh := range files {
		f, err := fh.Open()
		if err != nil {
			httpx.Error(w, http.StatusBadRequest, "could not read "+fh.Filename)
			return
		}
		data, err := io.ReadAll(f)
		f.Close()
		if err != nil {
			httpx.Error(w, http.StatusBadRequest, "could not read "+fh.Filename)
			return
		}
		ctype, ext, err := checkFile(kind, fh.Filename, data)
		if err != nil {
			httpx.Error(w, http.StatusBadRequest, fmt.Sprintf("%s: %v", fh.Filename, err))
			return
		}
		id := uuid.New()
		base := fmt.Sprintf("class-media/%s/%s/%s", sec.SchoolID, sectionID, kind)
		key := fmt.Sprintf("%s/%s%s", base, id, ext)
		if err := m.storage.Upload(key, ctype, data); err != nil {
			log.Printf("class media upload: %v", err)
			httpx.Error(w, http.StatusBadGateway, "could not store the file, please try again")
			return
		}
		var thumbKey *string
		if kind == KindPhoto {
			if thumb, err := makeThumbnail(data, 480); err == nil {
				k := fmt.Sprintf("%s/thumbs/%s.jpg", base, id)
				if err := m.storage.Upload(k, "image/jpeg", thumb); err == nil {
					thumbKey = &k
				}
			}
		}
		it := Item{
			ID: id, Kind: kind, Event: event, EventDate: eventDate, FileName: cleanName(fh.Filename),
			ContentType: ctype, SizeBytes: len(data), UploadedBy: &uploaderID,
			s3Key: key,
		}
		if thumbKey != nil {
			it.thumbKey = *thumbKey
		}
		if err := m.repo.Insert(r.Context(), sec, &it); err != nil {
			_ = m.storage.Delete(key)
			httpx.WriteServiceError(w, err)
			return
		}
		saved = append(saved, it)
	}
	httpx.JSON(w, http.StatusCreated, map[string]interface{}{"items": saved})
}

func (m *Module) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	it, err := m.repo.Get(r.Context(), id)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	acc, _, err := m.sectionAccess(r, it.SectionID)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	mine := it.UploadedBy != nil && it.UploadedBy.String() == httpx.ClaimsFromContext(r.Context()).Sub
	if !acc.manage && !(mine && acc.view) {
		httpx.Error(w, http.StatusForbidden, "access denied")
		return
	}
	if err := m.repo.Delete(r.Context(), id); err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	// The row is gone; S3 keeps a deleted file's old version for 90 days.
	if m.storage.Enabled() {
		if err := m.storage.Delete(it.s3Key); err != nil {
			log.Printf("class media delete: %v", err)
		}
		if it.thumbKey != "" {
			_ = m.storage.Delete(it.thumbKey)
		}
	}
	httpx.NoContent(w)
}

// ---- File checks ----

var documentTypes = map[string]string{
	".pdf":  "application/pdf",
	".doc":  "application/msword",
	".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
	".xls":  "application/vnd.ms-excel",
	".xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	".ppt":  "application/vnd.ms-powerpoint",
	".pptx": "application/vnd.openxmlformats-officedocument.presentationml.presentation",
	".txt":  "text/plain",
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".png":  "image/png",
}

var photoTypes = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/gif":  ".gif",
	"image/webp": ".webp",
}

var errType = errors.New("this kind of file can't be uploaded here")

// checkFile returns the stored content type and extension, judging photos
// by their actual bytes (not the name) so nothing else is served as an image.
func checkFile(kind, name string, data []byte) (string, string, error) {
	if len(data) == 0 {
		return "", "", errors.New("empty file")
	}
	ext := strings.ToLower(filepath.Ext(name))
	if kind == KindPhoto {
		if len(data) > maxPhotoBytes {
			return "", "", errors.New("photo larger than 15 MB")
		}
		sniffed := http.DetectContentType(data)
		if e, ok := photoTypes[sniffed]; ok {
			return sniffed, e, nil
		}
		// iPhone photos: HEIC isn't recognised by the sniffer; check its
		// "ftyp" box instead.
		if (ext == ".heic" || ext == ".heif") && len(data) > 12 && bytes.Equal(data[4:8], []byte("ftyp")) {
			return "image/heic", ext, nil
		}
		return "", "", errors.New("not a JPG, PNG, GIF, WEBP or HEIC photo")
	}
	if len(data) > maxDocumentBytes {
		return "", "", errors.New("document larger than 20 MB")
	}
	ctype, ok := documentTypes[ext]
	if !ok {
		return "", "", errType
	}
	return ctype, ext, nil
}

// cleanName keeps a file name safe to show and to offer for download.
func cleanName(n string) string {
	n = filepath.Base(strings.ReplaceAll(n, "\\", "/"))
	n = strings.Map(func(r rune) rune {
		if r < 32 || r == '"' {
			return -1
		}
		return r
	}, n)
	if len(n) > 150 {
		n = n[len(n)-150:]
	}
	if n == "" || n == "." {
		n = "file"
	}
	return n
}

// ---- Issued documents (archive) ----

// ListIssued lists a student's archived TCs, marksheets and report cards
// (see internal/platform/archive), newest first, with download links.
func (m *Module) ListIssued(w http.ResponseWriter, r *http.Request) {
	studentID, err := uuid.Parse(r.URL.Query().Get("student_id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "student_id required")
		return
	}
	claims := httpx.ClaimsFromContext(r.Context())
	docs, schoolID, err := m.repo.Issued(r.Context(), studentID)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	if claims.Role != "super_admin" && schoolID != nil && schoolID.String() != claims.SchoolID {
		httpx.Error(w, http.StatusForbidden, "access denied: school mismatch")
		return
	}
	for i := range docs {
		if m.storage.Enabled() {
			name := fmt.Sprintf("%s_%s_%s.pdf", docs[i].Kind, strings.ReplaceAll(docs[i].StudentName, " ", "_"), docs[i].IssuedAt.Format("2006-01-02"))
			docs[i].URL, _ = m.storage.PresignedDownloadURL(docs[i].s3Key, name, urlTTL)
		}
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{"items": docs})
}
