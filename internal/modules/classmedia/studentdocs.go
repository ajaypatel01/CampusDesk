package classmedia

// Student documents: scans/photos of a student's papers (bank passbook,
// Aadhaar card, certificates), stored privately in S3, for the owner,
// school admins and registrars. Details entered with a passbook, Aadhaar
// card, Samagra ID, caste or birth certificate are also written to the
// student's own record, so the profile stays the one source of truth; the
// document keeps only a summary (numbers as their last 4 digits).

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	apperr "github.com/ajaypatel01/CampusDesk/internal/platform/errors"
	"github.com/ajaypatel01/CampusDesk/internal/platform/httpx"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v4"
)

const maxStudentDocBytes = 10 << 20

var studentDocTypes = map[string]string{
	"bank_passbook":        "Bank Passbook",
	"aadhar_card":          "Aadhaar Card",
	"samagra_id":           "Samagra ID",
	"caste_certificate":    "Caste Certificate",
	"birth_certificate":    "Birth Certificate",
	"income_certificate":   "Income Certificate",
	"domicile_certificate": "Domicile Certificate",
	"transfer_certificate": "Transfer Certificate",
	"marksheet":            "Marksheet",
	"photo":                "Photo",
	"other":                "Other",
}

var (
	reIFSC    = regexp.MustCompile(`^[A-Z]{4}0[A-Z0-9]{6}$`)
	reAccount = regexp.MustCompile(`^[0-9]{9,18}$`)
	reAadhaar = regexp.MustCompile(`^[2-9][0-9]{11}$`)
	reSamagra = regexp.MustCompile(`^[0-9]{8,12}$`)
)

// StudentDoc is one uploaded document.
type StudentDoc struct {
	ID             uuid.UUID         `json:"id"`
	DocType        string            `json:"doc_type"`
	DocTypeLabel   string            `json:"doc_type_label"`
	Title          string            `json:"title"`
	Meta           map[string]string `json:"meta"`
	FileName       string            `json:"file_name"`
	ContentType    string            `json:"content_type"`
	SizeBytes      int               `json:"size_bytes"`
	UploadedByName string            `json:"uploaded_by_name"`
	CreatedAt      time.Time         `json:"created_at"`
	URL            string            `json:"url,omitempty"`
	s3Key          string
}

// studentSchool returns the student's school, or 403/404 for the caller.
func (m *Module) studentSchool(r *http.Request, studentID uuid.UUID) (uuid.UUID, error) {
	var schoolID uuid.UUID
	err := m.repo.pool.QueryRow(r.Context(), `SELECT school_id FROM students WHERE id = $1`, studentID).Scan(&schoolID)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, apperr.ErrNotFound
	}
	if err != nil {
		return uuid.Nil, err
	}
	claims := httpx.ClaimsFromContext(r.Context())
	if claims.Role != "super_admin" && schoolID.String() != claims.SchoolID {
		return uuid.Nil, fmt.Errorf("%w: school mismatch", apperr.ErrForbidden)
	}
	return schoolID, nil
}

func (m *Module) ListStudentDocs(w http.ResponseWriter, r *http.Request) {
	studentID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid student id")
		return
	}
	if _, err := m.studentSchool(r, studentID); err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	rows, err := m.repo.pool.Query(r.Context(), `
		SELECT d.id, d.doc_type, d.title, d.meta, d.file_name, d.content_type, d.size_bytes,
			COALESCE(TRIM(u.first_name || ' ' || u.last_name), ''), d.created_at, d.s3_key
		FROM student_documents d LEFT JOIN users u ON u.id = d.uploaded_by
		WHERE d.student_id = $1 ORDER BY d.created_at DESC`, studentID)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	defer rows.Close()
	docs := []StudentDoc{}
	for rows.Next() {
		var d StudentDoc
		var meta []byte
		if err := rows.Scan(&d.ID, &d.DocType, &d.Title, &meta, &d.FileName, &d.ContentType, &d.SizeBytes, &d.UploadedByName, &d.CreatedAt, &d.s3Key); err != nil {
			httpx.WriteServiceError(w, err)
			return
		}
		_ = json.Unmarshal(meta, &d.Meta)
		d.DocTypeLabel = studentDocTypes[d.DocType]
		if m.storage.Enabled() {
			d.URL, _ = m.storage.PresignedDownloadURL(d.s3Key, d.FileName, urlTTL)
		}
		docs = append(docs, d)
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{"items": docs, "types": studentDocTypes})
}

// UploadStudentDoc stores one document (multipart "file") with its type and
// details, and copies the details onto the student's record.
func (m *Module) UploadStudentDoc(w http.ResponseWriter, r *http.Request) {
	if !m.storage.Enabled() {
		httpx.Error(w, http.StatusServiceUnavailable, "file storage is not set up yet")
		return
	}
	studentID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid student id")
		return
	}
	schoolID, err := m.studentSchool(r, studentID)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxStudentDocBytes+(1<<20))
	if err := r.ParseMultipartForm(4 << 20); err != nil {
		httpx.Error(w, http.StatusBadRequest, "file too large (at most 10 MB)")
		return
	}
	docType := r.FormValue("doc_type")
	label, ok := studentDocTypes[docType]
	if !ok {
		httpx.Error(w, http.StatusBadRequest, "choose a document type")
		return
	}
	updates, meta, err := docDetails(docType, r.FormValue)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	title := strings.TrimSpace(r.FormValue("title"))
	if len(title) > 120 {
		title = title[:120]
	}
	f, fh, err := r.FormFile("file")
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "choose a file")
		return
	}
	data, err := io.ReadAll(f)
	f.Close()
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "could not read the file")
		return
	}
	ctype, ext, err := checkStudentDocFile(fh.Filename, data)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	id := uuid.New()
	key := fmt.Sprintf("student-docs/%s/%s/%s%s", schoolID, studentID, id, ext)
	if err := m.storage.Upload(key, ctype, data); err != nil {
		log.Printf("student doc upload: %v", err)
		httpx.Error(w, http.StatusBadGateway, "could not store the file, please try again")
		return
	}
	uploader, _ := uuid.Parse(httpx.ClaimsFromContext(r.Context()).Sub)
	metaJSON, _ := json.Marshal(meta)
	if err := m.saveStudentDoc(r.Context(), id, schoolID, studentID, docType, title, metaJSON, cleanName(fh.Filename), key, ctype, len(data), uploader, updates); err != nil {
		_ = m.storage.Delete(key)
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, map[string]interface{}{"id": id, "doc_type": docType, "doc_type_label": label, "updated_fields": keys(updates)})
}

func (m *Module) saveStudentDoc(ctx context.Context, id, schoolID, studentID uuid.UUID, docType, title string, meta []byte,
	fileName, key, ctype string, size int, uploader uuid.UUID, updates map[string]interface{}) error {
	tx, err := m.repo.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `
		INSERT INTO student_documents (id, school_id, student_id, doc_type, title, meta, file_name, s3_key, content_type, size_bytes, uploaded_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		id, schoolID, studentID, docType, title, meta, fileName, key, ctype, size, uploader); err != nil {
		return err
	}
	if len(updates) > 0 {
		// Column names come from docDetails' fixed list, never from input.
		sets, args := []string{}, []interface{}{studentID}
		for _, col := range keys(updates) {
			args = append(args, updates[col])
			sets = append(sets, fmt.Sprintf("%s = $%d", col, len(args)))
		}
		if _, err := tx.Exec(ctx, `UPDATE students SET `+strings.Join(sets, ", ")+`, updated_at = now() WHERE id = $1`, args...); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (m *Module) DeleteStudentDoc(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	var studentID uuid.UUID
	var key string
	err = m.repo.pool.QueryRow(r.Context(), `SELECT student_id, s3_key FROM student_documents WHERE id = $1`, id).Scan(&studentID, &key)
	if errors.Is(err, pgx.ErrNoRows) {
		httpx.Error(w, http.StatusNotFound, "not found")
		return
	}
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	if _, err := m.studentSchool(r, studentID); err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	if _, err := m.repo.pool.Exec(r.Context(), `DELETE FROM student_documents WHERE id = $1`, id); err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	if err := m.storage.Delete(key); err != nil {
		log.Printf("student doc delete: %v", err)
	}
	httpx.NoContent(w)
}

// docDetails validates the details sent with a document type and returns
// the student columns to update and the summary kept with the document.
func docDetails(docType string, get func(string) string) (map[string]interface{}, map[string]string, error) {
	val := func(k string) string { return strings.TrimSpace(get(k)) }
	updates := map[string]interface{}{}
	meta := map[string]string{}
	switch docType {
	case "bank_passbook":
		acct := strings.ReplaceAll(val("bank_account_number"), " ", "")
		ifsc := strings.ToUpper(val("bank_ifsc"))
		if acct == "" || ifsc == "" {
			return nil, nil, errors.New("enter the account number and IFSC from the passbook")
		}
		if !reAccount.MatchString(acct) {
			return nil, nil, errors.New("account number should be 9 to 18 digits")
		}
		if !reIFSC.MatchString(ifsc) {
			return nil, nil, errors.New("IFSC should look like SBIN0001234 (11 characters)")
		}
		updates["bank_account_number"], updates["bank_ifsc"] = acct, ifsc
		meta["account_last4"], meta["ifsc"] = acct[len(acct)-4:], ifsc
		for _, f := range []string{"bank_name", "bank_holder_name", "bank_branch"} {
			if v := val(f); v != "" {
				updates[f] = v
				meta[strings.TrimPrefix(f, "bank_")] = v
			}
		}
	case "aadhar_card":
		a := strings.NewReplacer(" ", "", "-", "").Replace(val("aadhar_number"))
		if !reAadhaar.MatchString(a) {
			return nil, nil, errors.New("Aadhaar number should be 12 digits")
		}
		updates["aadhar_number"] = a
		meta["aadhaar_last4"] = a[8:]
	case "samagra_id":
		s := strings.ReplaceAll(val("samagra_id"), " ", "")
		if !reSamagra.MatchString(s) {
			return nil, nil, errors.New("Samagra ID should be 8 to 12 digits")
		}
		updates["samagra_id"] = s
		meta["samagra_id"] = s
	case "caste_certificate":
		if c := val("caste"); c != "" {
			updates["caste"] = c
			meta["caste"] = c
		}
	case "birth_certificate":
		if d := val("date_of_birth"); d != "" {
			t, err := time.Parse("2006-01-02", d)
			if err != nil || t.After(time.Now()) {
				return nil, nil, errors.New("date of birth should be a past date (YYYY-MM-DD)")
			}
			updates["date_of_birth"] = t
			meta["date_of_birth"] = d
		}
	}
	return updates, meta, nil
}

// checkStudentDocFile accepts a PDF or a photo/scan, judged by content.
func checkStudentDocFile(name string, data []byte) (string, string, error) {
	if len(data) == 0 {
		return "", "", errors.New("empty file")
	}
	if len(data) > maxStudentDocBytes {
		return "", "", errors.New("file larger than 10 MB")
	}
	if bytes.HasPrefix(data, []byte("%PDF")) {
		return "application/pdf", ".pdf", nil
	}
	if ct, ext, err := checkFile(KindPhoto, name, data); err == nil {
		return ct, ext, nil
	}
	return "", "", errors.New("upload a PDF or a photo/scan (JPG, PNG, WEBP or HEIC)")
}

func keys(m map[string]interface{}) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	// Stable order, so the UPDATE's argument numbering is predictable.
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j] < out[j-1]; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}
