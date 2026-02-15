package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AuthProvider defines how the user authenticates
type AuthProvider string

const (
	AuthProviderEmail  AuthProvider = "email"
	AuthProviderGoogle AuthProvider = "google"
)

// User represents a registered user with multi-provider authentication
type User struct {
	ID              uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Name            string         `json:"name" gorm:"size:100;not null"`
	Email           string         `json:"email" gorm:"uniqueIndex;not null;size:255"`
	Password        string         `json:"-" gorm:"size:255"`                                  // NULL for Google OAuth users
	Avatar          string         `json:"avatar" gorm:"size:500;default:''"`
	AuthProvider    AuthProvider   `json:"auth_provider" gorm:"type:auth_provider;default:'email'"`
	GoogleID        *string        `json:"-" gorm:"uniqueIndex;size:255"`                      // Google's unique ID
	EmailVerifiedAt *time.Time     `json:"email_verified_at" gorm:"type:timestamptz"`          // NULL = not verified
	IsOnline        bool           `json:"is_online" gorm:"default:false"`
	LastSeen        *time.Time     `json:"last_seen"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `json:"-" gorm:"index"`
}

// IsEmailVerified checks if the user's email has been verified
func (u *User) IsEmailVerified() bool {
	return u.EmailVerifiedAt != nil
}

// UserResponse is the safe version of User for API responses
type UserResponse struct {
	ID            uuid.UUID    `json:"id"`
	Name          string       `json:"name"`
	Email         string       `json:"email"`
	Avatar        string       `json:"avatar"`
	AuthProvider  AuthProvider `json:"auth_provider"`
	EmailVerified bool         `json:"email_verified"`
	IsOnline      bool         `json:"is_online"`
	LastSeen      *time.Time   `json:"last_seen"`
}

// ToResponse converts User to safe UserResponse
func (u *User) ToResponse() UserResponse {
	return UserResponse{
		ID:            u.ID,
		Name:          u.Name,
		Email:         u.Email,
		Avatar:        u.Avatar,
		AuthProvider:  u.AuthProvider,
		EmailVerified: u.IsEmailVerified(),
		IsOnline:      u.IsOnline,
		LastSeen:      u.LastSeen,
	}
}
