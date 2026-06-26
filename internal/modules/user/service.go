package user

import (
	"context"
	"strings"

	"github.com/ajaypatel01/CampusDesk/internal/domain"
	apperr "github.com/ajaypatel01/CampusDesk/internal/platform/errors"
	"github.com/ajaypatel01/CampusDesk/internal/platform/httpx"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	repo      *Repository
	jwtSecret string
}

func NewService(repo *Repository, jwtSecret string) *Service {
	return &Service{repo: repo, jwtSecret: jwtSecret}
}

type CreateInput struct {
	SchoolID  *uuid.UUID       `json:"school_id"`
	Email     string           `json:"email"`
	Password  string           `json:"password"`
	FirstName string           `json:"first_name"`
	LastName  string           `json:"last_name"`
	Role      domain.UserRole  `json:"role"`
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
		IsActive:     true,
	}
	if err := s.repo.Create(ctx, u); err != nil {
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

func (s *Service) List(ctx context.Context, schoolID *uuid.UUID, limit, offset int) ([]domain.User, int, error) {
	users, total, err := s.repo.List(ctx, schoolID, limit, offset)
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
	if !u.IsActive {
		return nil, apperr.ErrUnauthorized
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(in.Password)); err != nil {
		return nil, apperr.ErrUnauthorized
	}
	u.PasswordHash = ""
	token, err := httpx.GenerateToken(u.ID.String(), string(u.Role), s.jwtSecret)
	if err != nil {
		return nil, err
	}
	return &LoginResponse{User: u, Token: token}, nil
}
