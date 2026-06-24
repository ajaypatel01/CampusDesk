package domain

import "github.com/google/uuid"

type Guardian struct {
	ID           uuid.UUID `json:"id"`
	FirstName    string    `json:"first_name"`
	LastName     string    `json:"last_name"`
	Email        string    `json:"email,omitempty"`
	Phone        string    `json:"phone,omitempty"`
	Relation     string    `json:"relation,omitempty"`
	AadharNumber string    `json:"aadhar_number,omitempty"`
	IsPrimary    bool      `json:"is_primary"`
	Timestamps
}

type StudentGuardian struct {
	StudentID  uuid.UUID `json:"student_id"`
	GuardianID uuid.UUID `json:"guardian_id"`
	IsPrimary  bool      `json:"is_primary"`
}
