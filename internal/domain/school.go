package domain

import "github.com/google/uuid"

type School struct {
	ID                  uuid.UUID `json:"id"`
	Name                string    `json:"name"`
	Code                string    `json:"code"`
	Address             string    `json:"address,omitempty"`
	Phone               string    `json:"phone,omitempty"`
	Email               string    `json:"email,omitempty"`
	WorkingDaysPerMonth int       `json:"working_days_per_month"`
	// DiceCode is the school's UDISE/DICE code, printed on report cards.
	// Distinct from tc_vouchers.dice_code, which is a per-voucher field.
	DiceCode string `json:"dice_code,omitempty"`
	Timestamps
}
