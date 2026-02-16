package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AttachmentType defines the type of attachment
type AttachmentType string

const (
	AttachmentTypeImage AttachmentType = "image"
	AttachmentTypeVideo AttachmentType = "video"
	AttachmentTypeFile  AttachmentType = "file"
	AttachmentTypeAudio AttachmentType = "audio"
)

// MessageAttachment represents a file attached to a message
type MessageAttachment struct {
	ID        uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	MessageID uuid.UUID      `json:"message_id" gorm:"type:uuid;index;not null"`
	Type      AttachmentType `json:"type" gorm:"type:varchar(20);not null"`
	URL       string         `json:"url" gorm:"size:1000;not null"`
	FileName  string         `json:"file_name" gorm:"size:255"`
	FileSize  int64          `json:"file_size"`
	MimeType  string         `json:"mime_type" gorm:"size:100"`
	Width     int            `json:"width,omitempty"`    // for images/videos
	Height    int            `json:"height,omitempty"`   // for images/videos
	Duration  float64        `json:"duration,omitempty"` // for audio/video (seconds)
	CreatedAt time.Time      `json:"created_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`

	// Relations
	Message Message `json:"-" gorm:"foreignKey:MessageID"`
}

// UploadResponse is returned after a successful file upload
type UploadResponse struct {
	URL      string `json:"url"`
	FileName string `json:"file_name"`
	FileSize int64  `json:"file_size"`
	MimeType string `json:"mime_type"`
}
