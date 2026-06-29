package domain

import "github.com/google/uuid"

type StaffProfile struct {
	ID                       uuid.UUID  `json:"id"`
	UserID                   uuid.UUID  `json:"user_id"`
	GuardianName             *string    `json:"guardian_name,omitempty"`
	AadharNumber             *string    `json:"aadhar_number,omitempty"`
	EducationQualification   *string    `json:"education_qualification,omitempty"`
	ProfessionalQualification *string   `json:"professional_qualification,omitempty"`
	Designation              *string    `json:"designation,omitempty"`
	Salary                   int        `json:"salary"`
	BankName                 *string    `json:"bank_name,omitempty"`
	BankIFSC                 *string    `json:"bank_ifsc,omitempty"`
	BankBranch               *string    `json:"bank_branch,omitempty"`
	BankAccountNumber        *string    `json:"bank_account_number,omitempty"`
	BankAccountHolder        *string    `json:"bank_account_holder,omitempty"`
	Phone                    *string    `json:"phone,omitempty"`
	Timestamps
}

type StaffMember struct {
	User
	Profile *StaffProfile `json:"profile,omitempty"`
}
