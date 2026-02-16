package model

import (
	"time"

	"github.com/google/uuid"
)

// UserDevice represents a user's device for push notifications
type UserDevice struct {
	ID           uuid.UUID `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID       uuid.UUID `json:"user_id" gorm:"not null;index"`
	FCMToken     string    `json:"fcm_token" gorm:"not null;uniqueIndex:idx_user_token"`
	DeviceType   string    `json:"device_type" gorm:"size:20;default:'unknown'"` // android, ios, web
	LastActiveAt time.Time `json:"last_active_at"`
	CreatedAt    time.Time `json:"created_at"`
}
