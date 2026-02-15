package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ConversationType defines whether the conversation is private or group
type ConversationType string

const (
	ConversationTypePrivate ConversationType = "private"
	ConversationTypeGroup   ConversationType = "group"
)

// Conversation represents a chat conversation (1-1 or group)
type Conversation struct {
	ID        uuid.UUID        `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Name      string           `json:"name" gorm:"size:100"`                              // group name, empty for private
	Type      ConversationType `json:"type" gorm:"type:varchar(20);default:'private'"`
	Avatar    string           `json:"avatar,omitempty" gorm:"size:500"`                   // group avatar
	CreatorID *uuid.UUID       `json:"creator_id,omitempty" gorm:"type:uuid"`              // group creator
	CreatedAt time.Time        `json:"created_at"`
	UpdatedAt time.Time        `json:"updated_at"`
	DeletedAt gorm.DeletedAt   `json:"-" gorm:"index"`

	// Relations
	Members      []ConversationMember `json:"members,omitempty" gorm:"foreignKey:ConversationID"`
	LastMessage  *Message             `json:"last_message,omitempty" gorm:"-"` // populated manually
}

// MemberRole defines the role of a member in a conversation
type MemberRole string

const (
	MemberRoleAdmin  MemberRole = "admin"
	MemberRoleMember MemberRole = "member"
)

// ConversationMember represents a user's membership in a conversation
type ConversationMember struct {
	ID             uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	ConversationID uuid.UUID      `json:"conversation_id" gorm:"type:uuid;uniqueIndex:idx_conv_user;not null"`
	UserID         uuid.UUID      `json:"user_id" gorm:"type:uuid;uniqueIndex:idx_conv_user;not null"`
	Role           MemberRole     `json:"role" gorm:"type:varchar(20);default:'member'"`
	JoinedAt       time.Time      `json:"joined_at"`
	LastReadAt     *time.Time     `json:"last_read_at,omitempty"`
	MutedUntil     *time.Time     `json:"muted_until,omitempty"`
	DeletedAt      gorm.DeletedAt `json:"-" gorm:"index"`

	// Relations
	User         User         `json:"user" gorm:"foreignKey:UserID"`
	Conversation Conversation `json:"-" gorm:"foreignKey:ConversationID"`
}
