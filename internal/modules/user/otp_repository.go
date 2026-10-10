package user

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ajaypatel01/CampusDesk/internal/domain"
	apperr "github.com/ajaypatel01/CampusDesk/internal/platform/errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v4"
)

const userColumns = `id, school_id, email, password_hash, first_name, last_name, role, status, is_active, token_version, phone_number, own_password, created_at, updated_at`

// phoneMatches compares users.phone_number by its last 10 digits, so a
// number saved as "+91 98765 43210" still matches "9876543210".
const phoneMatches = `phone_number IS NOT NULL AND right(regexp_replace(phone_number, '\D', '', 'g'), 10) = $1`

// ---- One-time codes ----

// CountOTPs counts codes sent to phone for purpose since each time.
func (r *Repository) CountOTPs(ctx context.Context, phone, purpose string, since1, since2 time.Time) (int, int, error) {
	var a, b int
	err := r.pool.QueryRow(ctx, `
		SELECT count(*) FILTER (WHERE created_at > $3), count(*) FILTER (WHERE created_at > $4)
		FROM otp_codes WHERE phone = $1 AND purpose = $2 AND created_at > LEAST($3, $4)`,
		phone, purpose, since1, since2,
	).Scan(&a, &b)
	return a, b, err
}

// CreateOTP stores a new code and retires any earlier unused ones.
func (r *Repository) CreateOTP(ctx context.Context, phone, purpose, codeHash string, expiresAt time.Time) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `
		UPDATE otp_codes SET used_at = now()
		WHERE phone = $1 AND purpose = $2 AND used_at IS NULL`, phone, purpose); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO otp_codes (phone, purpose, code_hash, expires_at) VALUES ($1, $2, $3, $4)`,
		phone, purpose, codeHash, expiresAt); err != nil {
		return err
	}
	// Old codes are kept a day for the send limits, then dropped.
	if _, err := tx.Exec(ctx, `DELETE FROM otp_codes WHERE created_at < now() - interval '2 days'`); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// TakeOTPAttempt counts one try against phone's live code and returns it,
// or apperr.ErrNotFound if there's no live code or its tries are used up.
func (r *Repository) TakeOTPAttempt(ctx context.Context, phone, purpose string, maxAttempts int) (int64, string, error) {
	var id int64
	var hash string
	err := r.pool.QueryRow(ctx, `
		UPDATE otp_codes SET attempts = attempts + 1
		WHERE id = (
			SELECT id FROM otp_codes
			WHERE phone = $1 AND purpose = $2 AND used_at IS NULL AND expires_at > now()
			ORDER BY created_at DESC LIMIT 1
		) AND attempts < $3
		RETURNING id, code_hash`, phone, purpose, maxAttempts,
	).Scan(&id, &hash)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, "", apperr.ErrNotFound
	}
	return id, hash, err
}

// MarkOTPUsed reports false if the code was already used.
func (r *Repository) MarkOTPUsed(ctx context.Context, id int64) (bool, error) {
	tag, err := r.pool.Exec(ctx, `UPDATE otp_codes SET used_at = now() WHERE id = $1 AND used_at IS NULL`, id)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

// ---- Finding who a phone number belongs to ----

// GetStaffByPhone finds a non-parent login whose verified phone is phone.
func (r *Repository) GetStaffByPhone(ctx context.Context, phone string) (*domain.User, error) {
	return r.scanOne(ctx, `SELECT `+userColumns+` FROM users WHERE `+phoneMatches+` AND role <> 'parent' ORDER BY created_at LIMIT 1`, phone)
}

// GetParentByPhone finds a parent login whose verified phone is phone.
func (r *Repository) GetParentByPhone(ctx context.Context, phone string) (*domain.User, error) {
	return r.scanOne(ctx, `SELECT `+userColumns+` FROM users WHERE `+phoneMatches+` AND role = 'parent' ORDER BY created_at LIMIT 1`, phone)
}

// guardianMatch is a guardian record whose phone field holds a number.
type guardianMatch struct {
	ID        uuid.UUID
	UserID    *uuid.UUID
	FirstName string
	LastName  string
	Phone     string
	SchoolID  uuid.UUID
}

// GuardiansByPhone returns the guardians of at least one student whose
// phone field contains phone (10 digits) as one of its numbers.
func (r *Repository) GuardiansByPhone(ctx context.Context, phone string) ([]guardianMatch, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT DISTINCT ON (g.id) g.id, g.user_id, g.first_name, g.last_name, g.phone, s.school_id
		FROM guardians g
		JOIN student_guardians sg ON sg.guardian_id = g.id
		JOIN students s ON s.id = sg.student_id
		WHERE regexp_replace(coalesce(g.phone, ''), '\D', '', 'g') LIKE '%' || $1 || '%'
		ORDER BY g.id, sg.is_primary DESC`, phone)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []guardianMatch
	for rows.Next() {
		var g guardianMatch
		if err := rows.Scan(&g.ID, &g.UserID, &g.FirstName, &g.LastName, &g.Phone, &g.SchoolID); err != nil {
			return nil, err
		}
		// The LIKE above is a rough filter; keep only real matches.
		for _, n := range mobileNumbers(g.Phone) {
			if n == phone {
				out = append(out, g)
				break
			}
		}
	}
	return out, rows.Err()
}

// EnsureParentLogin returns the parent login for these guardian records
// (all from one school), creating it if none exists, and links every
// record that isn't linked to a login yet. preferred, if set, is a parent
// login already known for this phone.
func (r *Repository) EnsureParentLogin(ctx context.Context, phone string, schoolID uuid.UUID, guardians []guardianMatch, preferred *uuid.UUID, passwordHash string) (*domain.User, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	ids := make([]uuid.UUID, 0, len(guardians))
	for _, g := range guardians {
		ids = append(ids, g.ID)
	}
	email := fmt.Sprintf("%s@parent.campusdesk.invalid", phone)

	var userID uuid.UUID
	if preferred != nil {
		userID = *preferred
	} else {
		err = tx.QueryRow(ctx, `
			SELECT u.id FROM users u
			WHERE u.role = 'parent'
			  AND (u.id IN (SELECT user_id FROM guardians WHERE id = ANY($1) AND user_id IS NOT NULL) OR u.email = $2)
			ORDER BY u.created_at LIMIT 1`, ids, email).Scan(&userID)
		if errors.Is(err, pgx.ErrNoRows) {
			g := guardians[0]
			err = tx.QueryRow(ctx, `
				INSERT INTO users (school_id, email, password_hash, first_name, last_name, role, status, is_active)
				VALUES ($1, $2, $3, $4, $5, 'parent', 'approved', true) RETURNING id`,
				schoolID, email, passwordHash, g.FirstName, g.LastName).Scan(&userID)
		}
		if err != nil {
			return nil, fmt.Errorf("find or create parent login: %w", err)
		}
	}
	if _, err := tx.Exec(ctx, `
		UPDATE guardians SET user_id = $1, updated_at = now()
		WHERE id = ANY($2) AND user_id IS NULL`, userID, ids); err != nil {
		return nil, fmt.Errorf("link guardians: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return r.GetByID(ctx, userID)
}
