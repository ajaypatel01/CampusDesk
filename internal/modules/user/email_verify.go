package user

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"html"
	"math/big"
	"net/mail"
	"strings"
	"time"

	"github.com/ajaypatel01/CampusDesk/internal/domain"
	"github.com/ajaypatel01/CampusDesk/internal/platform/database"
	apperr "github.com/ajaypatel01/CampusDesk/internal/platform/errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v4"
)

// ---- Email verification (logged in) ----
//
// A parent adds an email and confirms it with a 6-digit code sent to it;
// only then can they change their password (and later reset it by email).

const (
	emailCodeTTL      = 15 * time.Minute
	emailCodeAttempts = 5
	emailCodeLimit    = 3 // per 15 minutes
)

func cleanEmail(s string) (string, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	a, err := mail.ParseAddress(s)
	if err != nil || a.Address != s || !strings.Contains(s[strings.LastIndex(s, "@")+1:], ".") || strings.HasSuffix(s, ".invalid") {
		return "", fmt.Errorf("%w: enter a valid email address", apperr.ErrInvalidInput)
	}
	return s, nil
}

func (s *Service) emailCodeHash(userID uuid.UUID, email, code string) string {
	m := hmac.New(sha256.New, []byte(s.jwtSecret))
	m.Write([]byte("email|" + userID.String() + "|" + email + "|" + code))
	return hex.EncodeToString(m.Sum(nil))
}

// RequestEmailVerification emails a code to email for userID.
func (s *Service) RequestEmailVerification(ctx context.Context, userID uuid.UUID, email string) error {
	if s.emailClient == nil || !s.emailClient.Enabled() {
		return fmt.Errorf("%w: email is not set up on the server yet", apperr.ErrInvalidInput)
	}
	email, err := cleanEmail(email)
	if err != nil {
		return err
	}
	u, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	if other, err := s.repo.GetByEmail(ctx, email); err == nil && other.ID != userID {
		return fmt.Errorf("%w: that email is already used by another account", apperr.ErrConflict)
	}
	var recent int
	if err := s.repo.pool.QueryRow(ctx, `SELECT count(*) FROM email_codes WHERE user_id = $1 AND created_at > now() - interval '15 minutes'`, userID).Scan(&recent); err != nil {
		return err
	}
	if recent >= emailCodeLimit {
		return fmt.Errorf("%w: too many codes requested, please try again in 15 minutes", apperr.ErrTooMany)
	}
	n, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
	if err != nil {
		return err
	}
	code := fmt.Sprintf("%06d", n.Int64())
	if _, err := s.repo.pool.Exec(ctx, `UPDATE email_codes SET used_at = now() WHERE user_id = $1 AND used_at IS NULL`, userID); err != nil {
		return err
	}
	if _, err := s.repo.pool.Exec(ctx, `INSERT INTO email_codes (user_id, email, code_hash, expires_at) VALUES ($1, $2, $3, $4)`,
		userID, email, s.emailCodeHash(userID, email, code), time.Now().Add(emailCodeTTL)); err != nil {
		return err
	}
	name := strings.TrimSpace(u.FirstName + " " + u.LastName)
	body := fmt.Sprintf(`<p>Hello %s,</p><p>Your CampusDesk verification code is:</p>
<p style="font-size:28px;font-weight:700;letter-spacing:6px">%s</p>
<p>Enter it in the CampusDesk app or website to confirm this email. It expires in 15 minutes.</p>
<p>If you didn't ask for this, you can ignore this email.</p>`, html.EscapeString(name), code)
	if err := s.emailClient.SendText(email, name, "Your CampusDesk verification code", body); err != nil {
		return fmt.Errorf("send verification email: %w", err)
	}
	return nil
}

// ConfirmEmailVerification checks the code and saves email as userID's
// verified email.
func (s *Service) ConfirmEmailVerification(ctx context.Context, userID uuid.UUID, email, code string) (*domain.User, error) {
	email, err := cleanEmail(email)
	if err != nil {
		return nil, err
	}
	bad := fmt.Errorf("%w: invalid or expired code", apperr.ErrInvalidInput)
	var id int64
	var hash string
	err = s.repo.pool.QueryRow(ctx, `
		UPDATE email_codes SET attempts = attempts + 1
		WHERE id = (SELECT id FROM email_codes WHERE user_id = $1 AND email = $2 AND used_at IS NULL AND expires_at > now()
		            ORDER BY created_at DESC LIMIT 1)
		  AND attempts < $3
		RETURNING id, code_hash`, userID, email, emailCodeAttempts).Scan(&id, &hash)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, bad
	}
	if err != nil {
		return nil, err
	}
	if !hmac.Equal([]byte(hash), []byte(s.emailCodeHash(userID, email, strings.TrimSpace(code)))) {
		return nil, bad
	}
	if _, err := s.repo.pool.Exec(ctx, `UPDATE email_codes SET used_at = now() WHERE id = $1`, id); err != nil {
		return nil, err
	}
	if _, err := s.repo.pool.Exec(ctx, `UPDATE users SET email = $2, email_verified = true, updated_at = now() WHERE id = $1`, userID, email); err != nil {
		if errors.Is(database.MapError(err), apperr.ErrConflict) {
			return nil, fmt.Errorf("%w: that email is already used by another account", apperr.ErrConflict)
		}
		return nil, err
	}
	u, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	u.PasswordHash = ""
	return u, nil
}
