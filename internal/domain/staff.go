package domain

import "github.com/google/uuid"

type StaffProfile struct {
	ID                        uuid.UUID `json:"id"`
	UserID                    uuid.UUID `json:"user_id"`
	GuardianName              *string   `json:"guardian_name,omitempty"`
	AadharNumber              *string   `json:"aadhar_number,omitempty"`
	EducationQualification    *string   `json:"education_qualification,omitempty"`
	ProfessionalQualification *string   `json:"professional_qualification,omitempty"`
	Designation               *string   `json:"designation,omitempty"`
	Salary                    int       `json:"salary"` // derived: BasicSalary + HRA + SpecialAllowance + Bonus
	BasicSalary               int       `json:"basic_salary"`
	HRA                       int       `json:"hra"`
	SpecialAllowance          int       `json:"special_allowance"`
	Bonus                     int       `json:"bonus"`
	EPF                       int       `json:"epf"`
	ESIC                      int       `json:"esic"`
	AdditionalDeduction       int       `json:"additional_deduction"`
	AdditionalDeductionLabel  *string   `json:"additional_deduction_label,omitempty"`
	BankName                  *string   `json:"bank_name,omitempty"`
	BankIFSC                  *string   `json:"bank_ifsc,omitempty"`
	BankBranch                *string   `json:"bank_branch,omitempty"`
	BankAccountNumber         *string   `json:"bank_account_number,omitempty"`
	BankAccountHolder         *string   `json:"bank_account_holder,omitempty"`
	Phone                     *string   `json:"phone,omitempty"`
	StaffType                 *string   `json:"staff_type,omitempty"`
	CLQuotaPerYear            int       `json:"cl_quota_per_year"`
	Timestamps
}

type StaffMember struct {
	User
	Profile *StaffProfile `json:"profile,omitempty"`
}
