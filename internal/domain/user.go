package domain

import "github.com/google/uuid"

type User struct {
	ID           uuid.UUID  `json:"id"`
	SchoolID     *uuid.UUID `json:"school_id,omitempty"`
	Email        string     `json:"email"`
	FirstName    string     `json:"first_name"`
	LastName     string     `json:"last_name"`
	Role         UserRole   `json:"role"`
	PasswordHash string     `json:"-"`
	IsActive     bool       `json:"is_active"`
	Timestamps
}
