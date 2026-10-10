package classmedia

import (
	"context"
	"errors"
	"fmt"
	"time"

	apperr "github.com/ajaypatel01/CampusDesk/internal/platform/errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

// Section is a class section as the gallery shows it.
type Section struct {
	ID           uuid.UUID `json:"id"`
	SchoolID     uuid.UUID `json:"school_id"`
	GradeName    string    `json:"grade_name"`
	SectionName  string    `json:"section_name"`
	AcademicYear string    `json:"academic_year"`
	IsCurrent    bool      `json:"is_current_year"`
	// Children names the caller's children in this section (parents only).
	Children   string `json:"children,omitempty"`
	PhotoCount int    `json:"photo_count"`
	DocCount   int    `json:"document_count"`
}

// Item is one uploaded photo or document.
type Item struct {
	ID             uuid.UUID  `json:"id"`
	SectionID      uuid.UUID  `json:"class_section_id"`
	Kind           string     `json:"kind"`
	Event          string     `json:"event"`
	EventDate      *time.Time `json:"event_date,omitempty"`
	FileName       string     `json:"file_name"`
	ContentType    string     `json:"content_type"`
	SizeBytes      int        `json:"size_bytes"`
	UploadedBy     *uuid.UUID `json:"uploaded_by,omitempty"`
	UploadedByName string     `json:"uploaded_by_name,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	URL            string     `json:"url,omitempty"`
	ThumbURL       string     `json:"thumb_url,omitempty"`
	CanDelete      bool       `json:"can_delete"`
	s3Key          string
	thumbKey       string
}

const sectionSelect = `
	SELECT cs.id, cs.school_id, gl.name, cs.name, ay.name, ay.is_current, %s,
		(SELECT count(*) FROM class_media m WHERE m.class_section_id = cs.id AND m.kind = 'photo'),
		(SELECT count(*) FROM class_media m WHERE m.class_section_id = cs.id AND m.kind = 'document')
	FROM class_sections cs
	JOIN grade_levels gl ON gl.id = cs.grade_level_id
	JOIN academic_years ay ON ay.id = cs.academic_year_id`

const sectionOrder = ` ORDER BY ay.is_current DESC, ay.start_date DESC, gl.sort_order, gl.name, cs.name`

// isTeacherOf matches sections where $2 is the class or vice class teacher.
const isTeacherOf = `(cs.homeroom_teacher_id = $2 OR EXISTS (SELECT 1 FROM class_section_vice_teachers v WHERE v.class_section_id = cs.id AND v.user_id = $2))`

// parentChildren lists $2's children enrolled in cs.
const parentChildren = `(SELECT string_agg(DISTINCT s.first_name, ', ')
	FROM enrollments e
	JOIN students s ON s.id = e.student_id
	JOIN student_guardians sg ON sg.student_id = e.student_id
	JOIN guardians g ON g.id = sg.guardian_id
	WHERE e.class_section_id = cs.id AND g.user_id = $2 AND s.status <> 'duplicate')`

// Sections lists what role/userID may open in schoolID.
func (r *Repository) Sections(ctx context.Context, role string, userID, schoolID uuid.UUID) ([]Section, error) {
	var q string
	switch role {
	case "super_admin", "school_admin", "registrar":
		q = fmt.Sprintf(sectionSelect, "''") + ` WHERE cs.school_id = $1 AND $2::uuid IS NOT NULL` + sectionOrder
	case "teacher":
		q = fmt.Sprintf(sectionSelect, "''") + ` WHERE cs.school_id = $1 AND ` + isTeacherOf + sectionOrder
	case "parent":
		q = fmt.Sprintf(sectionSelect, "COALESCE("+parentChildren+", '')") + ` WHERE $1::uuid IS NOT NULL AND ` + parentChildren + ` IS NOT NULL` + sectionOrder
	default:
		return []Section{}, nil
	}
	rows, err := r.pool.Query(ctx, q, schoolID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Section{}
	for rows.Next() {
		var s Section
		if err := rows.Scan(&s.ID, &s.SchoolID, &s.GradeName, &s.SectionName, &s.AcademicYear, &s.IsCurrent, &s.Children, &s.PhotoCount, &s.DocCount); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *Repository) Section(ctx context.Context, id uuid.UUID) (*Section, error) {
	var s Section
	err := r.pool.QueryRow(ctx, `
		SELECT cs.id, cs.school_id, gl.name, cs.name, ay.name, ay.is_current
		FROM class_sections cs
		JOIN grade_levels gl ON gl.id = cs.grade_level_id
		JOIN academic_years ay ON ay.id = cs.academic_year_id
		WHERE cs.id = $1`, id).Scan(&s.ID, &s.SchoolID, &s.GradeName, &s.SectionName, &s.AcademicYear, &s.IsCurrent)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperr.ErrNotFound
	}
	return &s, err
}

func (r *Repository) TeachesSection(ctx context.Context, userID, sectionID uuid.UUID) (bool, error) {
	var ok bool
	err := r.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM class_sections cs WHERE cs.id = $1 AND `+isTeacherOf+`)`, sectionID, userID).Scan(&ok)
	return ok, err
}

func (r *Repository) ParentOfSection(ctx context.Context, userID, sectionID uuid.UUID) (bool, error) {
	var ok bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM enrollments e
			JOIN student_guardians sg ON sg.student_id = e.student_id
			JOIN guardians g ON g.id = sg.guardian_id
			WHERE e.class_section_id = $1 AND g.user_id = $2)`, sectionID, userID).Scan(&ok)
	return ok, err
}

const itemSelect = `
	SELECT m.id, m.class_section_id, m.kind, m.event, m.event_date, m.file_name, m.content_type, m.size_bytes,
		m.uploaded_by, COALESCE(TRIM(u.first_name || ' ' || u.last_name), ''), m.created_at, m.s3_key, COALESCE(m.thumb_key, '')
	FROM class_media m LEFT JOIN users u ON u.id = m.uploaded_by`

func scanItem(row pgx.Row) (Item, error) {
	var it Item
	err := row.Scan(&it.ID, &it.SectionID, &it.Kind, &it.Event, &it.EventDate, &it.FileName, &it.ContentType, &it.SizeBytes,
		&it.UploadedBy, &it.UploadedByName, &it.CreatedAt, &it.s3Key, &it.thumbKey)
	return it, err
}

// List returns a section's items of one kind, newest event first.
func (r *Repository) List(ctx context.Context, sectionID uuid.UUID, kind string) ([]Item, error) {
	rows, err := r.pool.Query(ctx, itemSelect+`
		WHERE m.class_section_id = $1 AND m.kind = $2
		ORDER BY COALESCE(m.event_date, m.created_at::date) DESC, m.event, m.created_at DESC`, sectionID, kind)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Item{}
	for rows.Next() {
		it, err := scanItem(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

func (r *Repository) Get(ctx context.Context, id uuid.UUID) (*Item, error) {
	it, err := scanItem(r.pool.QueryRow(ctx, itemSelect+` WHERE m.id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperr.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &it, nil
}

func (r *Repository) Insert(ctx context.Context, sec *Section, it *Item) error {
	var thumb *string
	if it.thumbKey != "" {
		thumb = &it.thumbKey
	}
	it.SectionID = sec.ID
	return r.pool.QueryRow(ctx, `
		INSERT INTO class_media (id, school_id, class_section_id, kind, event, event_date, file_name, s3_key, thumb_key, content_type, size_bytes, uploaded_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12) RETURNING created_at`,
		it.ID, sec.SchoolID, sec.ID, it.Kind, it.Event, it.EventDate, it.FileName, it.s3Key, thumb, it.ContentType, it.SizeBytes, it.UploadedBy,
	).Scan(&it.CreatedAt)
}

func (r *Repository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM class_media WHERE id = $1`, id)
	return err
}

// Issued is one archived document (see internal/platform/archive).
type Issued struct {
	ID           uuid.UUID `json:"id"`
	Kind         string    `json:"kind"`
	Title        string    `json:"title"`
	StudentName  string    `json:"student_name"`
	SizeBytes    int       `json:"size_bytes"`
	IssuedAt     time.Time `json:"issued_at"`
	IssuedByName string    `json:"issued_by_name"`
	URL          string    `json:"url,omitempty"`
	s3Key        string
}

// Issued returns studentID's archived documents and the student's school
// (nil when the student no longer exists and nothing is archived).
func (r *Repository) Issued(ctx context.Context, studentID uuid.UUID) ([]Issued, *uuid.UUID, error) {
	var school *uuid.UUID
	if err := r.pool.QueryRow(ctx, `SELECT school_id FROM students WHERE id = $1`, studentID).Scan(&school); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, err
	}
	rows, err := r.pool.Query(ctx, `
		SELECT d.id, d.kind, d.title, d.student_name, d.size_bytes, d.issued_at,
			COALESCE(TRIM(u.first_name || ' ' || u.last_name), ''), d.s3_key, d.school_id
		FROM issued_documents d LEFT JOIN users u ON u.id = d.issued_by
		WHERE d.student_id = $1 ORDER BY d.issued_at DESC`, studentID)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	out := []Issued{}
	for rows.Next() {
		var d Issued
		var sid uuid.UUID
		if err := rows.Scan(&d.ID, &d.Kind, &d.Title, &d.StudentName, &d.SizeBytes, &d.IssuedAt, &d.IssuedByName, &d.s3Key, &sid); err != nil {
			return nil, nil, err
		}
		if school == nil {
			school = &sid
		}
		out = append(out, d)
	}
	return out, school, rows.Err()
}
