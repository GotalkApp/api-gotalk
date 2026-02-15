package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// MessageType defines the type of message content
type MessageType string

const (
	MessageTypeText  MessageType = "text"
	MessageTypeImage MessageType = "image"
	MessageTypeFile  MessageType = "file"
)

// MessageStatus defines the delivery status of a message
type MessageStatus string

const (
	MessageStatusSent      MessageStatus = "sent"
	MessageStatusDelivered MessageStatus = "delivered"
	MessageStatusRead      MessageStatus = "read"
)

// Message represents a chat message
type Message struct {
	ID             uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	ConversationID uuid.UUID      `json:"conversation_id" gorm:"type:uuid;index;not null"`
	SenderID       uuid.UUID      `json:"sender_id" gorm:"type:uuid;index;not null"`
	Content        string         `json:"content" gorm:"type:text"`
	Type           MessageType    `json:"type" gorm:"type:varchar(20);default:'text'"`
	Status         MessageStatus  `json:"status" gorm:"type:varchar(20);default:'sent'"`
	FileURL        string         `json:"file_url,omitempty" gorm:"size:500"`
	FileName       string         `json:"file_name,omitempty" gorm:"size:255"`
	FileSize       int64          `json:"file_size,omitempty"`
	ReplyToID      *uuid.UUID     `json:"reply_to_id,omitempty" gorm:"type:uuid"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `json:"-" gorm:"index"`

	// Relations
	Sender       User          `json:"sender" gorm:"foreignKey:SenderID"`
	Conversation Conversation  `json:"-" gorm:"foreignKey:ConversationID"`
	ReplyTo      *Message      `json:"reply_to,omitempty" gorm:"foreignKey:ReplyToID"`
	ReadReceipts []ReadReceipt `json:"read_receipts,omitempty" gorm:"foreignKey:MessageID"`
}

// ReadReceipt tracks when a user reads a message
type ReadReceipt struct {
	ID        uuid.UUID `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	MessageID uuid.UUID `json:"message_id" gorm:"type:uuid;index;not null"`
	UserID    uuid.UUID `json:"user_id" gorm:"type:uuid;index;not null"`
	ReadAt    time.Time `json:"read_at" gorm:"not null"`

	// Relations
	Message Message `json:"-" gorm:"foreignKey:MessageID"`
	User    User    `json:"user" gorm:"foreignKey:UserID"`
}
