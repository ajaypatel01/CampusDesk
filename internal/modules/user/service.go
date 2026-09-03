package user

import (
	"context"
	"fmt"
	"strings"

	"github.com/ajaypatel01/CampusDesk/internal/domain"
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
	repo      *Repository
	jwtSecret string
}

func NewService(repo *Repository, jwtSecret string) *Service {
	return &Service{repo: repo, jwtSecret: jwtSecret}
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
