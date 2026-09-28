package user

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/ajaypatel01/CampusDesk/internal/domain"
	"github.com/ajaypatel01/CampusDesk/internal/platform/email"
	apperr "github.com/ajaypatel01/CampusDesk/internal/platform/errors"
	"github.com/ajaypatel01/CampusDesk/internal/platform/httpx"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// selfRegisterableRoles are the roles a user may request via public self-registration.
// super_admin accounts are provisioned only by another super_admin via the authenticated
// user-management endpoint, never through open registration.
var selfRegisterableRoles = map[domain.UserRole]bool{
	domain.RoleSchoolAdmin: true,
	domain.RoleTeacher:     true,
	domain.RoleRegistrar:   true,
	domain.RoleParent:      true,
}

type Service struct {
	repo        *Repository
	jwtSecret   string
	emailClient *email.Client
	frontendURL string
}

func NewService(repo *Repository, jwtSecret string, emailClient *email.Client, frontendURL string) *Service {
	return &Service{repo: repo, jwtSecret: jwtSecret, emailClient: emailClient, frontendURL: frontendURL}
}

type CreateInput struct {
	SchoolID  *uuid.UUID      `json:"school_id"`
	Email     string          `json:"email"`
	Password  string          `json:"password"`
	FirstName string          `json:"first_name"`
	LastName  string          `json:"last_name"`
	Role      domain.UserRole `json:"role"`
}

// RegisterInput is the public self-registration payload. Registered accounts are
// created with status "pending" and cannot log in until an admin approves them.
type RegisterInput struct {
	SchoolID  *uuid.UUID      `json:"school_id"`
	Email     string          `json:"email"`
	Password  string          `json:"password"`
	FirstName string          `json:"first_name"`
	LastName  string          `json:"last_name"`
	Role      domain.UserRole `json:"role"`
	// WardStudentCode and WardRelation are required when Role is "parent" — they
	// identify the student this parent should have portal access to.
	WardStudentCode string `json:"ward_student_code"`
	WardRelation    string `json:"ward_relation"`
}

type LoginInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResponse struct {
	User  *domain.User `json:"user"`
	Token string       `json:"token"`
}

func (s *Service) Create(ctx context.Context, in CreateInput) (*domain.User, error) {
	if strings.TrimSpace(in.Email) == "" || strings.TrimSpace(in.Password) == "" ||
		strings.TrimSpace(in.FirstName) == "" || strings.TrimSpace(in.LastName) == "" || in.Role == "" {
		return nil, apperr.ErrInvalidInput
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	u := &domain.User{
		SchoolID:     in.SchoolID,
		Email:        strings.TrimSpace(strings.ToLower(in.Email)),
		PasswordHash: string(hash),
		FirstName:    strings.TrimSpace(in.FirstName),
		LastName:     strings.TrimSpace(in.LastName),
		Role:         in.Role,
		Status:       domain.UserStatusApproved,
		IsActive:     true,
	}
	if err := s.repo.Create(ctx, u); err != nil {
		return nil, err
	}
	u.PasswordHash = ""
	return u, nil
}

// Register lets a new user request an account. The account is created inactive with
// status "pending" — it cannot log in until a super_admin/school_admin approves it.
func (s *Service) Register(ctx context.Context, in RegisterInput) (*domain.User, error) {
	if strings.TrimSpace(in.Email) == "" || strings.TrimSpace(in.Password) == "" ||
		strings.TrimSpace(in.FirstName) == "" || strings.TrimSpace(in.LastName) == "" || in.Role == "" {
		return nil, apperr.ErrInvalidInput
	}
	if len(in.Password) < 8 {
		return nil, fmt.Errorf("%w: password must be at least 8 characters", apperr.ErrInvalidInput)
	}
	if !selfRegisterableRoles[in.Role] {
		return nil, fmt.Errorf("%w: role is not available for self-registration", apperr.ErrInvalidInput)
	}
	if in.SchoolID == nil {
		return nil, fmt.Errorf("%w: school_id is required", apperr.ErrInvalidInput)
	}
	var wardStudentID uuid.UUID
	if in.Role == domain.RoleParent {
		code := strings.TrimSpace(in.WardStudentCode)
		if code == "" {
			return nil, fmt.Errorf("%w: ward_student_code is required for parent registration", apperr.ErrInvalidInput)
		}
		id, err := s.repo.FindStudentIDByCode(ctx, *in.SchoolID, code)
		if apperr.IsNotFound(err) {
			return nil, fmt.Errorf("%w: no student found with that scholar number at this school", apperr.ErrInvalidInput)
		}
		if err != nil {
			return nil, err
		}
		wardStudentID = id
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	u := &domain.User{
		SchoolID:     in.SchoolID,
		Email:        strings.TrimSpace(strings.ToLower(in.Email)),
		PasswordHash: string(hash),
		FirstName:    strings.TrimSpace(in.FirstName),
		LastName:     strings.TrimSpace(in.LastName),
		Role:         in.Role,
		Status:       domain.UserStatusPending,
		IsActive:     false,
	}
	if err := s.repo.Create(ctx, u); err != nil {
		return nil, err
	}
	if in.Role == domain.RoleParent {
		relation := strings.TrimSpace(in.WardRelation)
		if relation == "" {
			relation = "guardian"
		}
		if err := s.repo.LinkParentToStudent(ctx, u.ID, wardStudentID, u.FirstName, u.LastName, relation); err != nil {
			return nil, err
		}
	}
	u.PasswordHash = ""
	return u, nil
}

// Approve activates a pending registration, allowing the user to log in.
func (s *Service) Approve(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	u, err := s.repo.UpdateStatus(ctx, id, domain.UserStatusApproved, true)
	if err != nil {
		return nil, err
	}
	u.PasswordHash = ""
	return u, nil
}

// Reject declines a pending registration; the account remains inactive.
func (s *Service) Reject(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	u, err := s.repo.UpdateStatus(ctx, id, domain.UserStatusRejected, false)
	if err != nil {
		return nil, err
	}
	u.PasswordHash = ""
	return u, nil
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	u, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	u.PasswordHash = ""
	return u, nil
}

// UpdateInput edits a user's core profile fields and active status.
type UpdateInput struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
	IsActive  bool   `json:"is_active"`
}

func (s *Service) Update(ctx context.Context, id uuid.UUID, in UpdateInput) (*domain.User, error) {
	if strings.TrimSpace(in.FirstName) == "" || strings.TrimSpace(in.Email) == "" {
		return nil, apperr.ErrInvalidInput
	}
	u, err := s.repo.Update(ctx, id,
		strings.TrimSpace(in.FirstName), strings.TrimSpace(in.LastName),
		strings.TrimSpace(strings.ToLower(in.Email)), in.IsActive,
	)
	if err != nil {
		return nil, err
	}
	u.PasswordHash = ""
	return u, nil
}

func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}

func (s *Service) List(ctx context.Context, schoolID *uuid.UUID, status domain.UserStatus, limit, offset int) ([]domain.User, int, error) {
	users, total, err := s.repo.List(ctx, schoolID, status, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	for i := range users {
		users[i].PasswordHash = ""
	}
	return users, total, nil
}

func (s *Service) Login(ctx context.Context, in LoginInput) (*LoginResponse, error) {
	if strings.TrimSpace(in.Email) == "" || in.Password == "" {
		return nil, apperr.ErrInvalidInput
	}
	u, err := s.repo.GetByEmail(ctx, strings.ToLower(strings.TrimSpace(in.Email)))
	if err != nil {
		return nil, apperr.ErrUnauthorized
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(in.Password)); err != nil {
		return nil, apperr.ErrUnauthorized
	}
	switch u.Status {
	case domain.UserStatusPending:
		return nil, fmt.Errorf("%w: your registration is pending admin approval", apperr.ErrUnauthorized)
	case domain.UserStatusRejected:
		return nil, fmt.Errorf("%w: your registration was rejected", apperr.ErrUnauthorized)
	}
	if !u.IsActive {
		return nil, fmt.Errorf("%w: account is disabled", apperr.ErrUnauthorized)
	}
	u.PasswordHash = ""
	schoolID := ""
	if u.SchoolID != nil {
		schoolID = u.SchoolID.String()
	}
	token, err := httpx.GenerateToken(u.ID.String(), string(u.Role), schoolID, s.jwtSecret)
	if err != nil {
		return nil, err
	}
	return &LoginResponse{User: u, Token: token}, nil
}

// ---- Password reset ----

func hashResetToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// RequestPasswordReset always succeeds from the caller's point of view,
// whether or not the email belongs to an account -- never revealing which,
// so this can't be used to probe for registered emails. If it does belong to
// a real, non-pending/non-rejected account, a reset link is emailed.
func (s *Service) RequestPasswordReset(ctx context.Context, emailAddr string) error {
	emailAddr = strings.ToLower(strings.TrimSpace(emailAddr))
	if emailAddr == "" {
		return apperr.ErrInvalidInput
	}
	u, err := s.repo.GetByEmail(ctx, emailAddr)
	if err != nil {
		// Unknown email: report success anyway (see doc comment above).
		return nil
	}

	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return err
	}
	token := hex.EncodeToString(raw)

	t := &domain.PasswordResetToken{
		UserID:    u.ID,
		TokenHash: hashResetToken(token),
		ExpiresAt: time.Now().Add(30 * time.Minute),
	}
	if err := s.repo.CreatePasswordResetToken(ctx, t); err != nil {
		return err
	}

	if s.emailClient == nil || !s.emailClient.Enabled() {
		return nil // best-effort: no email configured, token still exists if needed manually
	}
	link := fmt.Sprintf("%s/reset-password?token=%s", strings.TrimRight(s.frontendURL, "/"), token)
	name := strings.TrimSpace(u.FirstName + " " + u.LastName)
	body := fmt.Sprintf(`<p>Hi %s,</p><p>Click the link below to reset your CampusDesk password. This link expires in 30 minutes and can only be used once.</p><p><a href="%s">%s</a></p><p>If you didn't request this, you can ignore this email.</p>`, name, link, link)
	if err := s.emailClient.SendText(u.Email, name, "Reset your CampusDesk password", body); err != nil {
		return fmt.Errorf("send reset email: %w", err)
	}
	return nil
}

// ConfirmPasswordReset validates the token (unused, unexpired) and sets the
// new password. A single generic error covers a wrong/reused/expired token
// so a token can't be probed for validity.
func (s *Service) ConfirmPasswordReset(ctx context.Context, token, newPassword string) error {
	if strings.TrimSpace(token) == "" || len(newPassword) < 6 {
		return apperr.ErrInvalidInput
	}
	t, err := s.repo.GetValidPasswordResetToken(ctx, hashResetToken(token))
	if err != nil {
		return fmt.Errorf("%w: this reset link is invalid or has expired", apperr.ErrInvalidInput)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	if err := s.repo.UpdatePasswordHash(ctx, t.UserID, string(hash)); err != nil {
		return err
	}
	return s.repo.MarkPasswordResetTokenUsed(ctx, t.ID)
}
