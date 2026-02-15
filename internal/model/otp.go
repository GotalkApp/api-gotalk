package model

import (
	"time"

	"github.com/google/uuid"
)

// OTPPurpose defines what the OTP code is used for
type OTPPurpose string

const (
	OTPPurposeEmailVerification OTPPurpose = "email_verification"
	OTPPurposePasswordReset     OTPPurpose = "password_reset"
)

// OTPCode represents a one-time password for email verification or password reset
type OTPCode struct {
	ID        uuid.UUID  `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID    uuid.UUID  `json:"user_id" gorm:"type:uuid;not null;index"`
	Code      string     `json:"-" gorm:"size:6;not null"`         // 6-digit numeric code
	Purpose   OTPPurpose `json:"purpose" gorm:"type:otp_purpose;default:'email_verification'"`
	ExpiresAt time.Time  `json:"expires_at" gorm:"not null"`       // When the code becomes invalid
	UsedAt    *time.Time `json:"used_at"`                          // NULL = not yet used
	CreatedAt time.Time  `json:"created_at"`

	// Relations
	User User `json:"-" gorm:"foreignKey:UserID"`
}

// IsExpired checks if the OTP code has expired
func (o *OTPCode) IsExpired() bool {
	return time.Now().After(o.ExpiresAt)
}

// IsUsed checks if the OTP code has already been used
func (o *OTPCode) IsUsed() bool {
	return o.UsedAt != nil
}

// IsValid checks if the OTP code can still be used
func (o *OTPCode) IsValid() bool {
	return !o.IsExpired() && !o.IsUsed()
}
