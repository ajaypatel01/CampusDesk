package domain

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID           uuid.UUID  `json:"id"`
	SchoolID     *uuid.UUID `json:"school_id,omitempty"`
	Email        string     `json:"email"`
	FirstName    string     `json:"first_name"`
	LastName     string     `json:"last_name"`
	Role         UserRole   `json:"role"`
	Status       UserStatus `json:"status"`
	PasswordHash string     `json:"-"`
	IsActive     bool       `json:"is_active"`
	// TokenVersion is embedded in every JWT minted for this user at login;
	// bumping it (LogoutEverywhere) invalidates every previously issued
	// token immediately, without a server-side session store.
	TokenVersion int `json:"-"`
	// PhoneNumber is set only once OTP-verified (see the user module's
	// RequestPhoneVerification/ConfirmPhoneVerification) -- never copied in
	// from a guardian/staff profile's contact number, since that isn't
	// necessarily the account owner's own phone.
	PhoneNumber *string `json:"phone_number,omitempty"`
	// OwnPassword is set once the user chooses their own password; until
	// then a parent can also log in with a child's first name + "@123".
	OwnPassword bool `json:"own_password"`
	Timestamps
}

// PasswordResetToken is a one-time, expiring self-service password reset
// link. TokenHash stores a hash of the emailed token, never the raw value --
// same reasoning as User.PasswordHash, so a DB read alone can't produce a
// usable reset link.
type PasswordResetToken struct {
	ID        uuid.UUID  `json:"id"`
	UserID    uuid.UUID  `json:"user_id"`
	TokenHash string     `json:"-"`
	ExpiresAt time.Time  `json:"expires_at"`
	UsedAt    *time.Time `json:"used_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}
