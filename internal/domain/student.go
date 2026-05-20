package domain

import (
	"time"

	"github.com/google/uuid"
)

type Student struct {
	ID          uuid.UUID     `json:"id"`
	SchoolID    uuid.UUID     `json:"school_id"`
	StudentCode string        `json:"student_code"`
	FirstName   string        `json:"first_name"`
	LastName    string        `json:"last_name"`
	DateOfBirth *time.Time    `json:"date_of_birth,omitempty"`
	Gender      string        `json:"gender,omitempty"`
	Email       string        `json:"email,omitempty"`
	Phone       string        `json:"phone,omitempty"`
	Address     string        `json:"address,omitempty"`
	Status      StudentStatus `json:"status"`
	Timestamps
}
