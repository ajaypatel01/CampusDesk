package domain

import (
	"time"

	"github.com/google/uuid"
)

type Student struct {
	ID                uuid.UUID     `json:"id"`
	SchoolID          uuid.UUID     `json:"school_id"`
	StudentCode       string        `json:"student_code"`
	FirstName         string        `json:"first_name"`
	LastName          string        `json:"last_name"`
	DateOfBirth       *time.Time    `json:"date_of_birth,omitempty"`
	Gender            string        `json:"gender,omitempty"`
	Email             string        `json:"email,omitempty"`
	Phone             string        `json:"phone,omitempty"`
	Address           string        `json:"address,omitempty"`
	AdmissionDate     *time.Time    `json:"admission_date,omitempty"`
	Caste             string        `json:"caste,omitempty"`
	Category          string        `json:"category,omitempty"`
	AadharNumber      string        `json:"aadhar_number,omitempty"`
	SamagraID         string        `json:"samagra_id,omitempty"`
	PenNumber         string        `json:"pen_number,omitempty"`
	AparID            string        `json:"apar_id,omitempty"`
	PreviousSchool    string        `json:"previous_school,omitempty"`
	BankName          string        `json:"bank_name,omitempty"`
	BankIFSC          string        `json:"bank_ifsc,omitempty"`
	BankAccountNumber string        `json:"bank_account_number,omitempty"`
	BankHolderName    string        `json:"bank_holder_name,omitempty"`
	BankBranch        string        `json:"bank_branch,omitempty"`
	Status            StudentStatus `json:"status"`
	Timestamps
}
