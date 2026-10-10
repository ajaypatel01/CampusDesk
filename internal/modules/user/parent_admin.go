package user

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/ajaypatel01/CampusDesk/internal/domain"
	apperr "github.com/ajaypatel01/CampusDesk/internal/platform/errors"
	"github.com/ajaypatel01/CampusDesk/internal/platform/httpx"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// ---- Parent logins (admin) ----
//
// A parent login is keyed by mobile number. Admins see every number linked
// to their school's students and can set a parent's password, or put it
// back to the default (any child's first name + "@123").

// ParentChild is a student linked to a parent mobile number.
type ParentChild struct {
	Name        string `json:"name"`
	StudentCode string `json:"student_code"`
	Class       string `json:"class"`
}

// ParentLogin is one parent mobile number and its login state.
type ParentLogin struct {
	Phone         string        `json:"phone"`
	Guardians     []string      `json:"guardians"`
	Children      []ParentChild `json:"children"`
	HasLogin      bool          `json:"has_login"`
	OwnPassword   bool          `json:"own_password"`
	Email         string        `json:"email,omitempty"`
	EmailVerified bool          `json:"email_verified"`
}

// ParentLogins lists schoolID's parent mobile numbers.
func (s *Service) ParentLogins(ctx context.Context, schoolID uuid.UUID) ([]ParentLogin, error) {
	rows, err := s.repo.pool.Query(ctx, `
		SELECT coalesce(g.phone, '') || ' ' || coalesce(s.phone, ''),
			trim(coalesce(g.first_name, '') || ' ' || coalesce(g.last_name, '')),
			g.user_id, trim(s.first_name || ' ' || coalesce(s.last_name, '')), s.student_code,
			coalesce(cls.name, '')
		FROM students s
		LEFT JOIN student_guardians sg ON sg.student_id = s.id
		LEFT JOIN guardians g ON g.id = sg.guardian_id
		LEFT JOIN LATERAL (
			SELECT gl.name || ' ' || cs.name AS name FROM enrollments e
			JOIN academic_years ay ON ay.id = e.academic_year_id AND ay.is_current
			JOIN class_sections cs ON cs.id = e.class_section_id
			JOIN grade_levels gl ON gl.id = cs.grade_level_id
			WHERE e.student_id = s.id LIMIT 1) cls ON true
		WHERE s.school_id = $1 AND s.status <> 'duplicate'`, schoolID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	byPhone := map[string]*ParentLogin{}
	users := map[string]uuid.UUID{}
	seenChild := map[string]bool{}
	seenGuardian := map[string]bool{}
	for rows.Next() {
		var phones, guardian, child, code, class string
		var userID *uuid.UUID
		if err := rows.Scan(&phones, &guardian, &userID, &child, &code, &class); err != nil {
			return nil, err
		}
		for _, n := range mobileNumbers(phones) {
			p := byPhone[n]
			if p == nil {
				p = &ParentLogin{Phone: n, Guardians: []string{}, Children: []ParentChild{}}
				byPhone[n] = p
			}
			if !seenChild[n+"|"+code] {
				seenChild[n+"|"+code] = true
				p.Children = append(p.Children, ParentChild{Name: child, StudentCode: code, Class: class})
			}
			if guardian != "" && !seenGuardian[n+"|"+guardian] {
				seenGuardian[n+"|"+guardian] = true
				p.Guardians = append(p.Guardians, guardian)
			}
			if _, ok := users[n]; !ok && userID != nil {
				users[n] = *userID
			}
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	ids := make([]uuid.UUID, 0, len(users))
	for _, id := range users {
		ids = append(ids, id)
	}
	type state struct {
		own, verified bool
		email         string
	}
	states := map[uuid.UUID]state{}
	if len(ids) > 0 {
		urows, err := s.repo.pool.Query(ctx, `SELECT id, own_password, coalesce(email, ''), email_verified FROM users WHERE id = ANY($1) AND role = 'parent'`, ids)
		if err != nil {
			return nil, err
		}
		defer urows.Close()
		for urows.Next() {
			var id uuid.UUID
			var st state
			if err := urows.Scan(&id, &st.own, &st.email, &st.verified); err != nil {
				return nil, err
			}
			states[id] = st
		}
		if err := urows.Err(); err != nil {
			return nil, err
		}
	}
	out := make([]ParentLogin, 0, len(byPhone))
	for n, p := range byPhone {
		if id, ok := users[n]; ok {
			if st, ok := states[id]; ok {
				p.HasLogin, p.OwnPassword, p.EmailVerified = true, st.own, st.verified
				if st.verified {
					p.Email = st.email
				}
			}
		}
		out = append(out, *p)
	}
	sort.Slice(out, func(i, j int) bool {
		a, b := strings.ToLower(out[i].Children[0].Name), strings.ToLower(out[j].Children[0].Name)
		if a != b {
			return a < b
		}
		return out[i].Phone < out[j].Phone
	})
	return out, nil
}

// SetParentPassword sets the password of the parent login for phone, or
// with reset puts it back to the default. The login is created if the
// parent has never logged in, and every session of it is logged out.
// schoolID limits the number to one school (nil: any school).
func (s *Service) SetParentPassword(ctx context.Context, schoolID *uuid.UUID, phone, password string, reset bool) error {
	phone, err := normalizePhone(phone)
	if err != nil {
		return err
	}
	if !reset && len(password) < 6 {
		return fmt.Errorf("%w: the password must be at least 6 characters", apperr.ErrInvalidInput)
	}
	t, err := s.findOTPLogin(ctx, phone, "parent")
	if err != nil {
		return fmt.Errorf("%w: no parent is linked to this number", apperr.ErrNotFound)
	}
	var parent *domain.User
	if len(t.guardians) > 0 {
		if schoolID != nil && t.schoolID != *schoolID {
			return fmt.Errorf("%w: no parent is linked to this number", apperr.ErrNotFound)
		}
		var preferred *uuid.UUID
		if t.user != nil {
			preferred = &t.user.ID
		} else if u, err := s.repo.ParentLoginForGuardians(ctx, guardianIDs(t.guardians)); err == nil {
			preferred = &u.ID
		} else if !errors.Is(err, apperr.ErrNotFound) {
			return err
		}
		hash, err := randomPasswordHash()
		if err != nil {
			return err
		}
		if parent, err = s.repo.EnsureParentLogin(ctx, phone, t.schoolID, t.guardians, preferred, hash); err != nil {
			return err
		}
	} else {
		parent = t.user
		if schoolID != nil && (parent.SchoolID == nil || *parent.SchoolID != *schoolID) {
			return fmt.Errorf("%w: no parent is linked to this number", apperr.ErrNotFound)
		}
	}
	if reset {
		hash, err := randomPasswordHash()
		if err != nil {
			return err
		}
		if _, err := s.repo.pool.Exec(ctx, `UPDATE users SET password_hash = $2, own_password = false, updated_at = now() WHERE id = $1`, parent.ID, hash); err != nil {
			return err
		}
	} else {
		hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		if err := s.repo.SetOwnPassword(ctx, parent.ID, string(hash)); err != nil {
			return err
		}
	}
	_, err = s.repo.BumpTokenVersion(ctx, parent.ID)
	return err
}

// parentSchool is the school whose parents the caller administers: their
// own, or for a super admin the school asked for (nil: any).
func parentSchool(r *http.Request, asked string) (*uuid.UUID, error) {
	school, err := adminSchool(r)
	if err != nil || school != nil || asked == "" {
		return school, err
	}
	id, err := uuid.Parse(asked)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid school_id", apperr.ErrInvalidInput)
	}
	return &id, nil
}

// ListParentLogins handles GET /parent-logins?school_id=.
func (h *Handler) ListParentLogins(w http.ResponseWriter, r *http.Request) {
	schoolID, err := parentSchool(r, r.URL.Query().Get("school_id"))
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	if schoolID == nil {
		httpx.Error(w, http.StatusBadRequest, "school_id is required")
		return
	}
	items, err := h.svc.ParentLogins(r.Context(), *schoolID)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{"items": items})
}

// SetParentPassword handles POST /parent-logins/password.
func (h *Handler) SetParentPassword(w http.ResponseWriter, r *http.Request) {
	var in struct {
		SchoolID string `json:"school_id"`
		Phone    string `json:"phone"`
		Password string `json:"password"`
		Reset    bool   `json:"reset"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid json body")
		return
	}
	schoolID, err := parentSchool(r, in.SchoolID)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	if err := h.svc.SetParentPassword(r.Context(), schoolID, in.Phone, in.Password, in.Reset); err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.NoContent(w)
}
