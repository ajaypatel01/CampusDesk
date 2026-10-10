package user

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/ajaypatel01/CampusDesk/internal/domain"
	"github.com/ajaypatel01/CampusDesk/internal/platform/database"
	apperr "github.com/ajaypatel01/CampusDesk/internal/platform/errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Create(ctx context.Context, u *domain.User) error {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO users (school_id, email, password_hash, first_name, last_name, role, status, is_active)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		RETURNING id, created_at, updated_at`,
		u.SchoolID, u.Email, u.PasswordHash, u.FirstName, u.LastName, u.Role, u.Status, u.IsActive,
	)
	if err := row.Scan(&u.ID, &u.CreatedAt, &u.UpdatedAt); err != nil {
		return database.MapError(err)
	}
	return nil
}

func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	return r.scanOne(ctx, `SELECT id, school_id, email, password_hash, first_name, last_name, role, status, is_active, token_version, phone_number, own_password, created_at, updated_at FROM users WHERE id=$1`, id)
}

// FindStudentIDByCode looks up a student by their scholar number (student_code) within
// a school, for linking a newly-registered parent account to their ward.
func (r *Repository) FindStudentIDByCode(ctx context.Context, schoolID uuid.UUID, code string) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.pool.QueryRow(ctx, `SELECT id FROM students WHERE school_id=$1 AND student_code=$2`, schoolID, code).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, apperr.ErrNotFound
	}
	if err != nil {
		return uuid.Nil, fmt.Errorf("find student by code: %w", err)
	}
	return id, nil
}

// LinkParentToStudent creates a guardian record carrying the parent user's id (so the
// portal can scope their access) and links it to their ward student. Not marked primary
// — this is a portal-access record, additive to whatever contact guardians already exist.
func (r *Repository) LinkParentToStudent(ctx context.Context, userID, studentID uuid.UUID, firstName, lastName, relation string) error {
	var guardianID uuid.UUID
	err := r.pool.QueryRow(ctx, `
		INSERT INTO guardians (first_name, last_name, relation, user_id)
		VALUES ($1,$2,$3,$4) RETURNING id`,
		firstName, lastName, relation, userID,
	).Scan(&guardianID)
	if err != nil {
		return fmt.Errorf("create parent guardian record: %w", err)
	}
	_, err = r.pool.Exec(ctx, `
		INSERT INTO student_guardians (student_id, guardian_id, is_primary)
		VALUES ($1,$2,false) ON CONFLICT DO NOTHING`,
		studentID, guardianID,
	)
	if err != nil {
		return fmt.Errorf("link parent to ward: %w", err)
	}
	return nil
}

func (r *Repository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	return r.scanOne(ctx, `SELECT id, school_id, email, password_hash, first_name, last_name, role, status, is_active, token_version, phone_number, own_password, created_at, updated_at FROM users WHERE email=$1`, email)
}

// GetByPhone looks up a user by their OTP-verified phone number -- used by
// OTP login and phone-based password reset. Unverified numbers (e.g. one
// copied from a staff/guardian profile) never land here, only ones that
// went through ConfirmPhoneVerification.
func (r *Repository) GetByPhone(ctx context.Context, phone string) (*domain.User, error) {
	return r.scanOne(ctx, `SELECT `+userColumns+` FROM users WHERE `+phoneMatches+` ORDER BY created_at LIMIT 1`, phone)
}

// SetPhoneNumber records a user's OTP-verified phone number. Returns
// apperr.ErrConflict if another account already has it (the partial unique
// index on users.phone_number) -- the caller should show a generic "that
// number is already in use" rather than which account has it.
func (r *Repository) SetPhoneNumber(ctx context.Context, userID uuid.UUID, phone string) error {
	tag, err := r.pool.Exec(ctx, `UPDATE users SET phone_number=$2, updated_at=NOW() WHERE id=$1`, userID, phone)
	if err != nil {
		return database.MapError(err)
	}
	if tag.RowsAffected() == 0 {
		return apperr.ErrNotFound
	}
	return nil
}

// UpdateStatus transitions a user's approval status (e.g. pending -> approved/rejected)
// and syncs is_active accordingly (only approved users may log in).
func (r *Repository) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.UserStatus, isActive bool) (*domain.User, error) {
	tag, err := r.pool.Exec(ctx, `UPDATE users SET status=$2, is_active=$3, updated_at=NOW() WHERE id=$1`, id, status, isActive)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, apperr.ErrNotFound
	}
	return r.GetByID(ctx, id)
}

// Update edits a user's core profile fields and active status.
func (r *Repository) Update(ctx context.Context, id uuid.UUID, firstName, lastName, email string, isActive bool) (*domain.User, error) {
	tag, err := r.pool.Exec(ctx, `
		UPDATE users SET first_name=$2, last_name=$3, email=$4, is_active=$5, updated_at=NOW()
		WHERE id=$1`, id, firstName, lastName, email, isActive)
	if err != nil {
		return nil, database.MapError(err)
	}
	if tag.RowsAffected() == 0 {
		return nil, apperr.ErrNotFound
	}
	return r.GetByID(ctx, id)
}

// Delete permanently removes a user account. Their homeroom assignments are
// cleared (ON DELETE SET NULL) and any staff profile is removed along with it
// (ON DELETE CASCADE) — see the FKs on class_sections and staff_profiles.
func (r *Repository) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM users WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return apperr.ErrNotFound
	}
	return nil
}

func (r *Repository) List(ctx context.Context, schoolID *uuid.UUID, status domain.UserStatus, limit, offset int) ([]domain.User, int, error) {
	var total int
	var rows pgx.Rows
	var err error

	where := make([]string, 0, 2)
	args := make([]interface{}, 0, 4)
	if schoolID != nil {
		args = append(args, *schoolID)
		where = append(where, fmt.Sprintf("school_id=$%d", len(args)))
	}
	if status != "" {
		args = append(args, status)
		where = append(where, fmt.Sprintf("status=$%d", len(args)))
	}
	whereClause := ""
	if len(where) > 0 {
		whereClause = "WHERE " + strings.Join(where, " AND ")
	}

	if err = r.pool.QueryRow(ctx, fmt.Sprintf(`SELECT COUNT(*) FROM users %s`, whereClause), args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	limitArgs := append(append([]interface{}{}, args...), limit, offset)
	rows, err = r.pool.Query(ctx, fmt.Sprintf(`
		SELECT id, school_id, email, password_hash, first_name, last_name, role, status, is_active, token_version, phone_number, own_password, created_at, updated_at
		FROM users %s ORDER BY is_active DESC, last_name, first_name LIMIT $%d OFFSET $%d`,
		whereClause, len(args)+1, len(args)+2), limitArgs...,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	return r.collect(rows, total)
}

func (r *Repository) collect(rows pgx.Rows, total int) ([]domain.User, int, error) {
	var users []domain.User
	for rows.Next() {
		u, err := scanRow(rows)
		if err != nil {
			return nil, 0, err
		}
		users = append(users, *u)
	}
	return users, total, rows.Err()
}

func (r *Repository) scanOne(ctx context.Context, q string, arg interface{}) (*domain.User, error) {
	row := r.pool.QueryRow(ctx, q, arg)
	u, err := scanRow(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperr.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan user: %w", err)
	}
	return u, nil
}

type scannable interface {
	Scan(dest ...interface{}) error
}

func scanRow(row scannable) (*domain.User, error) {
	var u domain.User
	err := row.Scan(&u.ID, &u.SchoolID, &u.Email, &u.PasswordHash, &u.FirstName, &u.LastName, &u.Role, &u.Status, &u.IsActive, &u.TokenVersion, &u.PhoneNumber, &u.OwnPassword, &u.CreatedAt, &u.UpdatedAt)
	return &u, err
}

// ---- Token version (log out everywhere) ----

// GetTokenVersion is the fast, single-column check every authenticated
// request makes (via the EnforceTokenVersion middleware) against the JWT's
// own "tv" claim.
func (r *Repository) GetTokenVersion(ctx context.Context, id uuid.UUID) (int, error) {
	var v int
	err := r.pool.QueryRow(ctx, `SELECT token_version FROM users WHERE id=$1`, id).Scan(&v)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, apperr.ErrNotFound
	}
	return v, err
}

// BumpTokenVersion invalidates every JWT issued to this user before now --
// their next request with an old token gets 401'd by EnforceTokenVersion,
// and the current device silently logs itself out via ClearToken same as a
// normal logout, so this is what "log out everywhere" actually does.
func (r *Repository) BumpTokenVersion(ctx context.Context, id uuid.UUID) (int, error) {
	var v int
	err := r.pool.QueryRow(ctx, `UPDATE users SET token_version = token_version + 1, updated_at = NOW() WHERE id=$1 RETURNING token_version`, id).Scan(&v)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, apperr.ErrNotFound
	}
	return v, err
}

// ---- Password reset ----

// UpdatePasswordHash sets a user's password hash directly -- used by the
// self-service password reset confirm step (and available for a future
// admin-reset action), bypassing the rest of Update's profile-field editing.
func (r *Repository) UpdatePasswordHash(ctx context.Context, userID uuid.UUID, hash string) error {
	tag, err := r.pool.Exec(ctx, `UPDATE users SET password_hash=$2, own_password=true, updated_at=NOW() WHERE id=$1`, userID, hash)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return apperr.ErrNotFound
	}
	return nil
}

func (r *Repository) CreatePasswordResetToken(ctx context.Context, t *domain.PasswordResetToken) error {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO password_reset_tokens (user_id, token_hash, expires_at)
		VALUES ($1,$2,$3) RETURNING id, created_at`,
		t.UserID, t.TokenHash, t.ExpiresAt,
	)
	return row.Scan(&t.ID, &t.CreatedAt)
}

// GetValidPasswordResetToken looks up an unused, unexpired token by its
// hash. Returns apperr.ErrNotFound for anything else (wrong/reused/expired
// token) -- the caller shows one generic "invalid or expired link" message
// either way, never distinguishing which, so a token can't be probed.
func (r *Repository) GetValidPasswordResetToken(ctx context.Context, tokenHash string) (*domain.PasswordResetToken, error) {
	var t domain.PasswordResetToken
	err := r.pool.QueryRow(ctx, `
		SELECT id, user_id, token_hash, expires_at, used_at, created_at
		FROM password_reset_tokens
		WHERE token_hash=$1 AND used_at IS NULL AND expires_at > NOW()`, tokenHash,
	).Scan(&t.ID, &t.UserID, &t.TokenHash, &t.ExpiresAt, &t.UsedAt, &t.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperr.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *Repository) MarkPasswordResetTokenUsed(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `UPDATE password_reset_tokens SET used_at=NOW() WHERE id=$1`, id)
	return err
}
