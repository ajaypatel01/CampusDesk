package user

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"html"
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
	otp         OTPSender
	frontendURL string
}

func NewService(repo *Repository, jwtSecret string, emailClient *email.Client, otp OTPSender, frontendURL string) *Service {
	return &Service{repo: repo, jwtSecret: jwtSecret, emailClient: emailClient, otp: otp, frontendURL: frontendURL}
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
	if err := checkLoginable(u); err != nil {
		return nil, err
	}
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
	body := fmt.Sprintf(`<p>Hi %s,</p><p>Click the link below to reset your CampusDesk password. This link expires in 30 minutes and can only be used once.</p><p><a href="%s">%s</a></p><p>If you didn't request this, you can ignore this email.</p>`,
		html.EscapeString(name), html.EscapeString(link), html.EscapeString(link))
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
	// A password reset is exactly the situation "log out everywhere" exists
	// for -- if someone else had a session open, a password change should
	// kick it out, not leave it valid for up to another 24h.
	if _, err := s.repo.BumpTokenVersion(ctx, t.UserID); err != nil {
		return err
	}
	return s.repo.MarkPasswordResetTokenUsed(ctx, t.ID)
}

// LogoutEverywhere invalidates every JWT issued to userID before now (see
// BumpTokenVersion) -- the caller's own current token is included, so the
// frontend must treat a successful call here exactly like a normal local
// logout (clear the stored token, redirect to login) rather than expecting
// to keep using the session that just called this.
func (s *Service) LogoutEverywhere(ctx context.Context, userID uuid.UUID) error {
	_, err := s.repo.BumpTokenVersion(ctx, userID)
	return err
}

// ---- Phone verification (self-service, logged in) ----
//
// A phone number is never copied in from a staff/guardian profile -- OTP
// verification is what actually establishes it belongs to the account
// owner, not just someone associated with the student/school.

// RequestPhoneVerification sends a WhatsApp code to phone. The number
// isn't saved yet -- only ConfirmPhoneVerification, after a correct code,
// actually sets users.phone_number.
func (s *Service) RequestPhoneVerification(ctx context.Context, phone string) error {
	phone, err := normalizePhone(phone)
	if err != nil {
		return err
	}
	if _, err := s.repo.GetByPhone(ctx, phone); err == nil {
		return fmt.Errorf("%w: that phone number is already in use on another account", apperr.ErrConflict)
	}
	return s.sendOTP(ctx, phone, otpPurposeVerifyPhone)
}

// ConfirmPhoneVerification checks the code and, if correct, sets
// userID's phone_number (stored as its 10 digits).
func (s *Service) ConfirmPhoneVerification(ctx context.Context, userID uuid.UUID, phone, otp string) error {
	phone, err := normalizePhone(phone)
	if err != nil {
		return err
	}
	if err := s.checkOTP(ctx, phone, otpPurposeVerifyPhone, otp); err != nil {
		return fmt.Errorf("%w: invalid or expired code", apperr.ErrInvalidInput)
	}
	return s.repo.SetPhoneNumber(ctx, userID, phone)
}

// ---- OTP login ----
//
// Staff log in with the phone they verified in Settings. Parents log in
// with the phone the school has on a guardian record; their login is
// created on first use and linked to every guardian record with that phone.
// audience is "staff" (staff app), "parent" (parent app) or "" (web: staff
// first, then parent) -- it decides which login a teacher who is also a
// parent gets.

// otpLoginTarget is who a phone number logs in as.
type otpLoginTarget struct {
	user      *domain.User    // an existing login, if known
	guardians []guardianMatch // parent: guardian records to link
	schoolID  uuid.UUID       // parent: the school those records are from
}

var errNoPhoneAccount = fmt.Errorf("%w: no account found for this number -- staff can add their number in Settings, parents should ask the school office to update it", apperr.ErrUnauthorized)

func (s *Service) findOTPLogin(ctx context.Context, phone, audience string) (*otpLoginTarget, error) {
	if audience != "" && audience != "staff" && audience != "parent" {
		return nil, apperr.ErrInvalidInput
	}
	if audience != "parent" {
		if u, err := s.repo.GetStaffByPhone(ctx, phone); err == nil {
			return &otpLoginTarget{user: u}, nil
		}
		if audience == "staff" {
			return nil, errNoPhoneAccount
		}
	}
	t := &otpLoginTarget{}
	if u, err := s.repo.GetParentByPhone(ctx, phone); err == nil {
		t.user = u
	}
	matches, err := s.repo.GuardiansByPhone(ctx, phone)
	if err != nil {
		return nil, err
	}
	if len(matches) > 0 {
		// A login belongs to one school: use the school with the most
		// matching records (normally the only one).
		count := map[uuid.UUID]int{}
		for _, g := range matches {
			count[g.SchoolID]++
			if count[g.SchoolID] > count[t.schoolID] {
				t.schoolID = g.SchoolID
			}
		}
		for _, g := range matches {
			if g.SchoolID == t.schoolID {
				t.guardians = append(t.guardians, g)
			}
		}
	}
	if t.user == nil && len(t.guardians) == 0 {
		return nil, errNoPhoneAccount
	}
	return t, nil
}

// RequestOTPLogin sends a login code on WhatsApp if phone belongs to a
// usable account -- unlike RequestPasswordReset's email flow, this
// deliberately does NOT report uniform success for an unknown number,
// since each message costs money; the tradeoff is a number can be probed
// for whether it's registered, same as the password login's error message.
func (s *Service) RequestOTPLogin(ctx context.Context, phone, audience string) error {
	if !s.otp.enabled() {
		return errOTPNotConfigured
	}
	phone, err := normalizePhone(phone)
	if err != nil {
		return err
	}
	t, err := s.findOTPLogin(ctx, phone, audience)
	if err != nil {
		return err
	}
	if t.user != nil {
		if err := checkLoginable(t.user); err != nil {
			return err
		}
	}
	return s.sendOTP(ctx, phone, otpPurposeLogin)
}

// VerifyOTPLogin mints a JWT exactly like Login, after checking the code --
// same pending/rejected/disabled account checks apply.
func (s *Service) VerifyOTPLogin(ctx context.Context, phone, otp, audience string) (*LoginResponse, error) {
	phone, err := normalizePhone(phone)
	if err != nil {
		return nil, err
	}
	t, err := s.findOTPLogin(ctx, phone, audience)
	if err != nil {
		return nil, err
	}
	if t.user != nil {
		if err := checkLoginable(t.user); err != nil {
			return nil, err
		}
	}
	if err := s.checkOTP(ctx, phone, otpPurposeLogin, otp); err != nil {
		return nil, err
	}
	u := t.user
	if len(t.guardians) > 0 {
		var preferred *uuid.UUID
		if u != nil {
			preferred = &u.ID
		}
		// The login has no usable password; the parent can set one with
		// "forgot password" later if they add an email.
		hash, err := randomPasswordHash()
		if err != nil {
			return nil, err
		}
		if u, err = s.repo.EnsureParentLogin(ctx, phone, t.schoolID, t.guardians, preferred, hash); err != nil {
			return nil, err
		}
		if err := checkLoginable(u); err != nil {
			return nil, err
		}
	}
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

func randomPasswordHash() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	h, err := bcrypt.GenerateFromPassword([]byte(hex.EncodeToString(b)[:64]), bcrypt.DefaultCost)
	return string(h), err
}

// checkLoginable applies the same pending/rejected/disabled checks Login
// uses, factored out so OTP login enforces them identically.
func checkLoginable(u *domain.User) error {
	switch u.Status {
	case domain.UserStatusPending:
		return fmt.Errorf("%w: your registration is pending admin approval", apperr.ErrUnauthorized)
	case domain.UserStatusRejected:
		return fmt.Errorf("%w: your registration was rejected", apperr.ErrUnauthorized)
	}
	if !u.IsActive {
		return fmt.Errorf("%w: account is disabled", apperr.ErrUnauthorized)
	}
	return nil
}

// ---- Phone-OTP password reset ----
//
// An alternative to the email-link flow for an account with a verified
// phone number -- not a replacement for it.

func (s *Service) RequestPasswordResetOTP(ctx context.Context, phone string) error {
	if !s.otp.enabled() {
		return errOTPNotConfigured
	}
	phone, err := normalizePhone(phone)
	if err != nil {
		return err
	}
	if _, err := s.repo.GetByPhone(ctx, phone); err != nil {
		return nil // unknown number: report success anyway, same reasoning as the email flow
	}
	return s.sendOTP(ctx, phone, otpPurposeReset)
}

func (s *Service) ConfirmPasswordResetOTP(ctx context.Context, phone, otp, newPassword string) error {
	if len(newPassword) < 6 {
		return apperr.ErrInvalidInput
	}
	phone, err := normalizePhone(phone)
	if err != nil {
		return err
	}
	u, err := s.repo.GetByPhone(ctx, phone)
	if err != nil {
		return fmt.Errorf("%w: this OTP is invalid or has expired", apperr.ErrInvalidInput)
	}
	if err := s.checkOTP(ctx, phone, otpPurposeReset, otp); err != nil {
		return fmt.Errorf("%w: this OTP is invalid or has expired", apperr.ErrInvalidInput)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	if err := s.repo.UpdatePasswordHash(ctx, u.ID, string(hash)); err != nil {
		return err
	}
	_, err = s.repo.BumpTokenVersion(ctx, u.ID)
	return err
}
