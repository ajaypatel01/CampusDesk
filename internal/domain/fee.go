package domain

import (
	"time"

	"github.com/google/uuid"
)

type FeeStructure struct {
	ID               uuid.UUID `json:"id"`
	SchoolID         uuid.UUID `json:"school_id"`
	AcademicYearID   uuid.UUID `json:"academic_year_id"`
	GradeLevelID     uuid.UUID `json:"grade_level_id"`
	TuitionFeeAnnual int       `json:"tuition_fee_annual"`
	NumInstallments  int       `json:"num_installments"`
	VanFeeAnnual     int       `json:"van_fee_annual"`
	Timestamps
}

type FeeInstallmentPlan struct {
	ID                uuid.UUID `json:"id"`
	FeeStructureID    uuid.UUID `json:"fee_structure_id"`
	InstallmentNumber int       `json:"installment_number"`
	Label             string    `json:"label"`
	Amount            int       `json:"amount"`
	DueDate           time.Time `json:"due_date"`
	Timestamps
}

type StudentFeeAccount struct {
	ID               uuid.UUID `json:"id"`
	StudentID        uuid.UUID `json:"student_id"`
	SchoolID         uuid.UUID `json:"school_id"`
	AcademicYearID   uuid.UUID `json:"academic_year_id"`
	FeeStructureID   uuid.UUID `json:"fee_structure_id"`
	TuitionFee       int       `json:"tuition_fee"`
	DiscountAmount   int       `json:"discount_amount"`
	DiscountReason   string    `json:"discount_reason,omitempty"`
	PreviousYearDues int       `json:"previous_year_dues"`
	VanFee           int       `json:"van_fee"`
	IsRTE            bool      `json:"is_rte"`
	Timestamps
}

type FeePayment struct {
	ID                  uuid.UUID   `json:"id"`
	StudentFeeAccountID uuid.UUID   `json:"student_fee_account_id"`
	FeeType             FeeType     `json:"fee_type"`
	InstallmentNumber   *int        `json:"installment_number,omitempty"`
	Amount              int         `json:"amount"`
	PaymentDate         time.Time   `json:"payment_date"`
	PaymentMode         PaymentMode `json:"payment_mode"`
	ReferenceNumber     string      `json:"reference_number,omitempty"`
	Notes               string      `json:"notes,omitempty"`
	Voided              bool        `json:"voided"`
	Timestamps
}
