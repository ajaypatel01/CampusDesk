package domain

import "github.com/google/uuid"

type School struct {
	ID      uuid.UUID `json:"id"`
	Name    string    `json:"name"`
	Code    string    `json:"code"`
	Address string    `json:"address,omitempty"`
	Phone   string    `json:"phone,omitempty"`
	Email   string    `json:"email,omitempty"`
	Timestamps
}
