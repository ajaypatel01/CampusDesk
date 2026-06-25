package communications

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v4/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

type Broadcast struct {
	ID            uuid.UUID  `json:"id"`
	SchoolID      uuid.UUID  `json:"school_id"`
	Title         string     `json:"title"`
	Message       string     `json:"message"`
	Target        string     `json:"target"`
	GradeLevelID  *uuid.UUID `json:"grade_level_id,omitempty"`
	TemplateName  string     `json:"template_name,omitempty"`
	TemplateLang  string     `json:"template_lang,omitempty"`
	IsTemplate    bool       `json:"is_template"`
	SentBy        *uuid.UUID `json:"sent_by,omitempty"`
	TotalCount    int        `json:"total_count"`
	SentCount     int        `json:"sent_count"`
	FailedCount   int        `json:"failed_count"`
	Status        string     `json:"status"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

type BroadcastRecipient struct {
	ID           uuid.UUID  `json:"id"`
	BroadcastID  uuid.UUID  `json:"broadcast_id"`
	Phone        string     `json:"phone"`
	Name         string     `json:"name,omitempty"`
	Status       string     `json:"status"`
	ErrorMessage string     `json:"error_message,omitempty"`
	SentAt       *time.Time `json:"sent_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}

type PhoneEntry struct {
	Phone string
	Name  string
}

func (r *Repository) CreateBroadcast(ctx context.Context, b *Broadcast) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO broadcasts
			(school_id, title, message, target, grade_level_id, template_name, template_lang,
			 is_template, sent_by, total_count, status)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,'sending')
		RETURNING id, status, created_at, updated_at`,
		b.SchoolID, b.Title, b.Message, b.Target, b.GradeLevelID, b.TemplateName, b.TemplateLang,
		b.IsTemplate, b.SentBy, b.TotalCount,
	).Scan(&b.ID, &b.Status, &b.CreatedAt, &b.UpdatedAt)
}

func (r *Repository) UpdateBroadcastCounts(ctx context.Context, id uuid.UUID, sentCount, failedCount int, status string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE broadcasts
		SET sent_count=$2, failed_count=$3, status=$4, updated_at=NOW()
		WHERE id=$1`, id, sentCount, failedCount, status)
	return err
}

func (r *Repository) AddRecipient(ctx context.Context, rec *BroadcastRecipient) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO broadcast_recipients (broadcast_id, phone, name, status, error_message, sent_at)
		VALUES ($1,$2,$3,$4,$5,$6)
		RETURNING id, created_at`,
		rec.BroadcastID, rec.Phone, rec.Name, rec.Status, rec.ErrorMessage, rec.SentAt,
	).Scan(&rec.ID, &rec.CreatedAt)
}

func (r *Repository) ListBroadcasts(ctx context.Context, schoolID uuid.UUID) ([]Broadcast, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, school_id, title, message, target, grade_level_id, template_name, template_lang,
			is_template, sent_by, total_count, sent_count, failed_count, status, created_at, updated_at
		FROM broadcasts WHERE school_id=$1
		ORDER BY created_at DESC`, schoolID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Broadcast
	for rows.Next() {
		var b Broadcast
		if err := rows.Scan(&b.ID, &b.SchoolID, &b.Title, &b.Message, &b.Target, &b.GradeLevelID,
			&b.TemplateName, &b.TemplateLang, &b.IsTemplate, &b.SentBy,
			&b.TotalCount, &b.SentCount, &b.FailedCount, &b.Status, &b.CreatedAt, &b.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, b)
	}
	return items, rows.Err()
}

func (r *Repository) ListRecipients(ctx context.Context, broadcastID uuid.UUID) ([]BroadcastRecipient, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, broadcast_id, phone, COALESCE(name,''), status, COALESCE(error_message,''), sent_at, created_at
		FROM broadcast_recipients WHERE broadcast_id=$1
		ORDER BY created_at`, broadcastID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []BroadcastRecipient
	for rows.Next() {
		var rec BroadcastRecipient
		if err := rows.Scan(&rec.ID, &rec.BroadcastID, &rec.Phone, &rec.Name,
			&rec.Status, &rec.ErrorMessage, &rec.SentAt, &rec.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, rec)
	}
	return items, rows.Err()
}

// LookupGradeParentPhones returns phone numbers of guardians whose students are enrolled in the given grade/year.
func (r *Repository) LookupGradeParentPhones(ctx context.Context, schoolID, gradeLevelID, yearID uuid.UUID) ([]PhoneEntry, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT DISTINCT
			COALESCE(g.mobile, g.phone, '') AS phone,
			g.first_name || ' ' || g.last_name AS name
		FROM guardians g
		JOIN students s ON s.id = g.student_id
		JOIN enrollments e ON e.student_id = s.id
		LEFT JOIN class_sections cs ON cs.id = e.class_section_id
		WHERE s.school_id = $1
		  AND e.academic_year_id = $3
		  AND cs.grade_level_id = $2
		  AND (g.mobile <> '' OR g.phone <> '')
		  AND s.is_active = TRUE`, schoolID, gradeLevelID, yearID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPhones(rows)
}

// LookupAllParentPhones returns phone numbers of all active guardians in a school.
func (r *Repository) LookupAllParentPhones(ctx context.Context, schoolID uuid.UUID) ([]PhoneEntry, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT DISTINCT
			COALESCE(g.mobile, g.phone, '') AS phone,
			g.first_name || ' ' || g.last_name AS name
		FROM guardians g
		JOIN students s ON s.id = g.student_id
		WHERE s.school_id = $1
		  AND (g.mobile <> '' OR g.phone <> '')
		  AND s.is_active = TRUE`, schoolID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPhones(rows)
}

// LookupStaffPhones returns phone numbers of all active staff in a school.
func (r *Repository) LookupStaffPhones(ctx context.Context, schoolID uuid.UUID) ([]PhoneEntry, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT DISTINCT
			COALESCE(u.phone, '') AS phone,
			u.first_name || ' ' || u.last_name AS name
		FROM users u
		WHERE u.school_id = $1
		  AND u.phone <> ''
		  AND u.is_active = TRUE`, schoolID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPhones(rows)
}

func scanPhones(rows interface{ Next() bool; Scan(...interface{}) error; Err() error; Close() }) ([]PhoneEntry, error) {
	defer rows.Close()
	var items []PhoneEntry
	for rows.Next() {
		var e PhoneEntry
		if err := rows.Scan(&e.Phone, &e.Name); err != nil {
			return nil, err
		}
		if e.Phone != "" {
			items = append(items, e)
		}
	}
	return items, rows.Err()
}
