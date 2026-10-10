package user

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ajaypatel01/CampusDesk/internal/domain"
	apperr "github.com/ajaypatel01/CampusDesk/internal/platform/errors"
	"github.com/ajaypatel01/CampusDesk/internal/platform/httpx"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v4"
	"golang.org/x/crypto/bcrypt"
)

// ---- Mobile number + password login ----
//
// The login box takes an email or a mobile number. With a mobile number:
// staff log in with their own password (number verified in Settings);
// parents log in with any linked child's first name + "@123" until they
// set their own password (users.own_password), and their login is created
// on first use exactly like WhatsApp OTP login (see findOTPLogin).

const (
	defaultPasswordSuffix = "@123"
	// At most loginFailLimit wrong passwords per number per loginFailWindow.
	loginFailLimit  = 5
	loginFailWindow = 15 * time.Minute
)

var errBadLogin = fmt.Errorf("%w: invalid credentials", apperr.ErrUnauthorized)

func (s *Service) phoneLogin(ctx context.Context, phone, password string) (*LoginResponse, error) {
	fails, err := s.repo.CountLoginFailures(ctx, phone, time.Now().Add(-loginFailWindow))
	if err != nil {
		return nil, err
	}
	if fails >= loginFailLimit {
		return nil, fmt.Errorf("%w: too many wrong passwords for this number, please try again in 15 minutes", apperr.ErrTooMany)
	}
	u, err := s.matchPhonePassword(ctx, phone, password)
	if err != nil {
		if errors.Is(err, errBadLogin) {
			_ = s.repo.RecordLoginFailure(ctx, phone)
		}
		return nil, err
	}
	if err := checkLoginable(u); err != nil {
		return nil, err
	}
	return s.issueToken(u)
}

// matchPhonePassword returns the login phone+password opens, or errBadLogin.
func (s *Service) matchPhonePassword(ctx context.Context, phone, password string) (*domain.User, error) {
	// Staff (and parents) who verified this number use their own password.
	if u, err := s.repo.GetStaffByPhone(ctx, phone); err == nil && passwordMatches(u, password) {
		return u, nil
	}
	t, err := s.findOTPLogin(ctx, phone, "parent")
	if err != nil {
		return nil, errBadLogin
	}
	parent := t.user
	if parent == nil && len(t.guardians) > 0 {
		if parent, err = s.repo.ParentLoginForGuardians(ctx, guardianIDs(t.guardians)); err != nil && !errors.Is(err, apperr.ErrNotFound) {
			return nil, err
		}
	}
	ok := parent != nil && passwordMatches(parent, password)
	if !ok && (parent == nil || !parent.OwnPassword) && len(t.guardians) > 0 {
		names, err := s.repo.StudentFirstNames(ctx, guardianIDs(t.guardians))
		if err != nil {
			return nil, err
		}
		ok = isDefaultPassword(password, names)
	}
	if !ok {
		return nil, errBadLogin
	}
	if len(t.guardians) == 0 {
		return parent, nil
	}
	// Create the login on first use and link every guardian record with
	// this number to it.
	var preferred *uuid.UUID
	if parent != nil {
		preferred = &parent.ID
	}
	hash, err := randomPasswordHash()
	if err != nil {
		return nil, err
	}
	return s.repo.EnsureParentLogin(ctx, phone, t.schoolID, t.guardians, preferred, hash)
}

// isDefaultPassword reports whether password is one of the children's
// first names + "@123". The name part ignores case and spaces, and a first
// name of several words also matches by its first word ("Ajay Kumar" ->
// "Ajay@123" or "AjayKumar@123").
func isDefaultPassword(password string, firstNames []string) bool {
	if !strings.HasSuffix(password, defaultPasswordSuffix) {
		return false
	}
	typed := strings.ToLower(strings.TrimSuffix(password, defaultPasswordSuffix))
	if typed == "" {
		return false
	}
	for _, n := range firstNames {
		words := strings.Fields(strings.ToLower(n))
		if len(words) == 0 {
			continue
		}
		if typed == words[0] || typed == strings.Join(words, "") {
			return true
		}
	}
	return false
}

func passwordMatches(u *domain.User, password string) bool {
	return u.PasswordHash != "" && bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)) == nil
}

func guardianIDs(gs []guardianMatch) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(gs))
	for _, g := range gs {
		ids = append(ids, g.ID)
	}
	return ids
}

func (s *Service) issueToken(u *domain.User) (*LoginResponse, error) {
	u.PasswordHash = ""
	schoolID := ""
	if u.SchoolID != nil {
		schoolID = u.SchoolID.String()
	}
	token, err := httpx.GenerateToken(u.ID.String(), string(u.Role), schoolID, u.TokenVersion, s.jwtSecret)
	if err != nil {
		return nil, err
	}
	return &LoginResponse{User: u, Token: token}, nil
}

// ---- Change password (logged in) ----

// ChangePassword sets the caller's own password after checking the current
// one -- for a parent still on the default, the default counts as current.
// Every other session is logged out; the returned token keeps this one.
func (s *Service) ChangePassword(ctx context.Context, userID uuid.UUID, current, next string) (*LoginResponse, error) {
	if len(next) < 6 {
		return nil, fmt.Errorf("%w: the new password must be at least 6 characters", apperr.ErrInvalidInput)
	}
	u, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	ok := passwordMatches(u, current)
	if !ok && u.Role == domain.RoleParent && !u.OwnPassword {
		names, err := s.repo.WardFirstNames(ctx, userID)
		if err != nil {
			return nil, err
		}
		ok = isDefaultPassword(current, names)
	}
	if !ok {
		return nil, fmt.Errorf("%w: the current password is wrong", apperr.ErrInvalidInput)
	}
	if u.Role == domain.RoleParent {
		names, err := s.repo.WardFirstNames(ctx, userID)
		if err != nil {
			return nil, err
		}
		if isDefaultPassword(next, names) {
			return nil, fmt.Errorf("%w: choose a password different from the default (child's name@123)", apperr.ErrInvalidInput)
		}
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(next), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	if err := s.repo.SetOwnPassword(ctx, userID, string(hash)); err != nil {
		return nil, err
	}
	if _, err := s.repo.BumpTokenVersion(ctx, userID); err != nil {
		return nil, err
	}
	u, err = s.repo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return s.issueToken(u)
}

// ---- Repository ----

func (r *Repository) CountLoginFailures(ctx context.Context, phone string, since time.Time) (int, error) {
	var n int
	err := r.pool.QueryRow(ctx, `SELECT count(*) FROM login_failures WHERE phone = $1 AND at > $2`, phone, since).Scan(&n)
	return n, err
}

func (r *Repository) RecordLoginFailure(ctx context.Context, phone string) error {
	_, err := r.pool.Exec(ctx, `
		WITH ins AS (INSERT INTO login_failures (phone) VALUES ($1))
		DELETE FROM login_failures WHERE at < now() - interval '1 day'`, phone)
	return err
}

// ParentLoginForGuardians returns the parent login already linked to any of
// these guardian records, or apperr.ErrNotFound.
func (r *Repository) ParentLoginForGuardians(ctx context.Context, ids []uuid.UUID) (*domain.User, error) {
	var id uuid.UUID
	err := r.pool.QueryRow(ctx, `
		SELECT u.id FROM users u JOIN guardians g ON g.user_id = u.id
		WHERE g.id = ANY($1) AND u.role = 'parent'
		ORDER BY u.created_at LIMIT 1`, ids).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperr.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return r.GetByID(ctx, id)
}

// StudentFirstNames returns the first names of the students linked to these
// guardian records.
func (r *Repository) StudentFirstNames(ctx context.Context, guardianIDs []uuid.UUID) ([]string, error) {
	return r.names(ctx, `
		SELECT DISTINCT s.first_name FROM students s
		JOIN student_guardians sg ON sg.student_id = s.id
		WHERE sg.guardian_id = ANY($1)`, guardianIDs)
}

// WardFirstNames returns the first names of a parent login's children.
func (r *Repository) WardFirstNames(ctx context.Context, userID uuid.UUID) ([]string, error) {
	return r.names(ctx, `
		SELECT DISTINCT s.first_name FROM students s
		JOIN student_guardians sg ON sg.student_id = s.id
		JOIN guardians g ON g.id = sg.guardian_id
		WHERE g.user_id = $1`, userID)
}

func (r *Repository) names(ctx context.Context, q string, arg interface{}) ([]string, error) {
	rows, err := r.pool.Query(ctx, q, arg)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

// SetOwnPassword stores a password the user chose themselves.
func (r *Repository) SetOwnPassword(ctx context.Context, userID uuid.UUID, hash string) error {
	_, err := r.pool.Exec(ctx, `UPDATE users SET password_hash = $2, own_password = true, updated_at = now() WHERE id = $1`, userID, hash)
	return err
}
