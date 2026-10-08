// Package auditlog serves the audit_log table (filled by a database trigger,
// see migration 000039): who added, changed or deleted which record, when.
package auditlog

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ajaypatel01/CampusDesk/internal/platform/httpx"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v4/pgxpool"
)

type Module struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Module { return &Module{pool: pool} }

func (m *Module) Name() string { return "auditlog" }

// Mount: the owner sees every school; a school admin only their own.
func (m *Module) Mount(r chi.Router) {
	r.With(httpx.RequireRole("super_admin", "school_admin")).Get("/audit-log", m.List)
}

type Entry struct {
	ID        int64           `json:"id"`
	At        time.Time       `json:"at"`
	Table     string          `json:"table"`
	Action    string          `json:"action"`
	RowID     string          `json:"row_id,omitempty"`
	SchoolID  *uuid.UUID      `json:"school_id,omitempty"`
	UserID    *uuid.UUID      `json:"user_id,omitempty"`
	UserName  string          `json:"user_name,omitempty"`
	UserEmail string          `json:"user_email,omitempty"`
	UserRole  string          `json:"user_role,omitempty"`
	Request   string          `json:"request,omitempty"`
	IP        string          `json:"ip,omitempty"`
	UserAgent string          `json:"user_agent,omitempty"`
	OldData   json.RawMessage `json:"old_data,omitempty"`
	NewData   json.RawMessage `json:"new_data,omitempty"`
}

var ist = time.FixedZone("IST", 5*60*60+30*60)

// List returns log entries, newest first. Filters (all optional):
// school_id, user_id, table, action, row_id, from/to (YYYY-MM-DD, India
// time, inclusive), limit (max 200), offset.
func (m *Module) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	claims := httpx.ClaimsFromContext(r.Context())
	where := []string{"TRUE"}
	args := []interface{}{}
	add := func(cond string, v interface{}) {
		args = append(args, v)
		where = append(where, fmt.Sprintf(cond, len(args)))
	}

	schoolID := q.Get("school_id")
	if claims == nil {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if claims.Role != "super_admin" {
		schoolID = claims.SchoolID // a school admin only ever sees their own school
	}
	if schoolID != "" {
		id, err := uuid.Parse(schoolID)
		if err != nil {
			httpx.Error(w, http.StatusBadRequest, "invalid school_id")
			return
		}
		add("a.school_id = $%d", id)
	}
	if v := q.Get("user_id"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			httpx.Error(w, http.StatusBadRequest, "invalid user_id")
			return
		}
		add("a.user_id = $%d", id)
	}
	if v := q.Get("table"); v != "" {
		add("a.table_name = $%d", v)
	}
	if v := strings.ToUpper(q.Get("action")); v != "" {
		if v != "INSERT" && v != "UPDATE" && v != "DELETE" {
			httpx.Error(w, http.StatusBadRequest, "action must be INSERT, UPDATE or DELETE")
			return
		}
		add("a.action = $%d", v)
	}
	if v := q.Get("row_id"); v != "" {
		add("a.row_id = $%d", v)
	}
	for _, f := range []struct {
		param, cond string
		addDay      bool
	}{{"from", "a.at >= $%d", false}, {"to", "a.at < $%d", true}} {
		v := q.Get(f.param)
		if v == "" {
			continue
		}
		d, err := time.ParseInLocation("2006-01-02", v, ist)
		if err != nil {
			httpx.Error(w, http.StatusBadRequest, f.param+" must be YYYY-MM-DD")
			return
		}
		if f.addDay {
			d = d.AddDate(0, 0, 1)
		}
		add(f.cond, d)
	}
	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	offset, _ := strconv.Atoi(q.Get("offset"))
	if offset < 0 {
		offset = 0
	}

	items, total, err := m.list(r.Context(), strings.Join(where, " AND "), args, limit, offset)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{"items": items, "total": total, "limit": limit, "offset": offset})
}

func (m *Module) list(ctx context.Context, where string, args []interface{}, limit, offset int) ([]Entry, int, error) {
	var total int
	if err := m.pool.QueryRow(ctx, `SELECT COUNT(*) FROM audit_log a WHERE `+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args = append(args, limit, offset)
	rows, err := m.pool.Query(ctx, fmt.Sprintf(`
		SELECT a.id, a.at, a.table_name, a.action, COALESCE(a.row_id, ''), a.school_id, a.user_id,
			COALESCE(TRIM(u.first_name || ' ' || u.last_name), ''), COALESCE(u.email, ''), COALESCE(a.user_role, ''),
			COALESCE(a.request, ''), COALESCE(a.ip, ''), COALESCE(a.user_agent, ''), a.old_data, a.new_data
		FROM audit_log a LEFT JOIN users u ON u.id = a.user_id
		WHERE %s ORDER BY a.at DESC, a.id DESC LIMIT $%d OFFSET $%d`, where, len(args)-1, len(args)), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := []Entry{}
	for rows.Next() {
		var e Entry
		var oldData, newData []byte
		if err := rows.Scan(&e.ID, &e.At, &e.Table, &e.Action, &e.RowID, &e.SchoolID, &e.UserID,
			&e.UserName, &e.UserEmail, &e.UserRole, &e.Request, &e.IP, &e.UserAgent, &oldData, &newData); err != nil {
			return nil, 0, err
		}
		e.OldData, e.NewData = oldData, newData
		items = append(items, e)
	}
	return items, total, rows.Err()
}
