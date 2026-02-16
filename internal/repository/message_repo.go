package repository

import (
	"github.com/google/uuid"
	"github.com/quocanhngo/gotalk/internal/model"
	"gorm.io/gorm"
)

// MessageRepository handles database operations for Message
type MessageRepository struct {
	db *gorm.DB
}

func NewMessageRepository(db *gorm.DB) *MessageRepository {
	return &MessageRepository{db: db}
}

// Create inserts a new message
func (r *MessageRepository) Create(msg *model.Message) error {
	return r.db.Create(msg).Error
}

// FindByID finds a message by ID
func (r *MessageRepository) FindByID(id uuid.UUID) (*model.Message, error) {
	var msg model.Message
	err := r.db.
		Preload("Sender").
		Preload("Attachments").
		Where("id = ?", id).
		First(&msg).Error
	if err != nil {
		return nil, err
	}
	return &msg, nil
}

// GetConversationMessages returns paginated messages for a conversation (cursor-based)
func (r *MessageRepository) GetConversationMessages(conversationID uuid.UUID, before *uuid.UUID, limit int) ([]model.Message, error) {
	messages := []model.Message{}
	query := r.db.
		Preload("Sender").
		Preload("Attachments").
		Where("conversation_id = ?", conversationID).
		Order("created_at DESC").
		Limit(limit)

	// Cursor-based pagination: get messages before a specific message
	if before != nil {
		var beforeMsg model.Message
		if err := r.db.Select("created_at").Where("id = ?", before).First(&beforeMsg).Error; err != nil {
			return nil, err
		}
		query = query.Where("created_at < ?", beforeMsg.CreatedAt)
	}

	err := query.Find(&messages).Error
	return messages, err
}

// GetLastMessage returns the most recent message in a conversation
func (r *MessageRepository) GetLastMessage(conversationID uuid.UUID) (*model.Message, error) {
	var msg model.Message
	err := r.db.
		Preload("Sender").
		Where("conversation_id = ?", conversationID).
		Order("created_at DESC").
		First(&msg).Error
	if err != nil {
		return nil, err
	}
	return &msg, nil
}

// GetUnreadMessages returns unread messages for a user in a conversation
func (r *MessageRepository) GetUnreadMessages(conversationID, userID uuid.UUID) ([]model.Message, error) {
	messages := []model.Message{}

	subQuery := r.db.Table("conversation_members").
		Select("COALESCE(last_read_at, '0001-01-01')").
		Where("conversation_id = ? AND user_id = ?", conversationID, userID)

	err := r.db.
		Preload("Sender").
		Where("conversation_id = ? AND sender_id != ?", conversationID, userID).
		Where("created_at > (?)", subQuery).
		Order("created_at ASC").
		Find(&messages).Error
	return messages, err
}

// CountUnread counts unread messages for a user in a conversation
func (r *MessageRepository) CountUnread(conversationID, userID uuid.UUID) (int64, error) {
	var count int64

	subQuery := r.db.Table("conversation_members").
		Select("COALESCE(last_read_at, '0001-01-01')").
		Where("conversation_id = ? AND user_id = ?", conversationID, userID)

	err := r.db.Model(&model.Message{}).
		Where("conversation_id = ? AND sender_id != ?", conversationID, userID).
		Where("created_at > (?)", subQuery).
		Count(&count).Error
	return count, err
}

// CreateAttachment inserts a new message attachment
func (r *MessageRepository) CreateAttachment(att *model.MessageAttachment) error {
	return r.db.Create(att).Error
}
