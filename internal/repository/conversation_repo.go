package repository

import (
	"github.com/google/uuid"
	"github.com/quocanhngo/gotalk/internal/model"
	"gorm.io/gorm"
)

// ConversationRepository handles database operations for Conversation
type ConversationRepository struct {
	db *gorm.DB
}

func NewConversationRepository(db *gorm.DB) *ConversationRepository {
	return &ConversationRepository{db: db}
}

// Create creates a new conversation with members
func (r *ConversationRepository) Create(conv *model.Conversation) error {
	return r.db.Create(conv).Error
}

// FindByID finds a conversation by ID with members
func (r *ConversationRepository) FindByID(id uuid.UUID) (*model.Conversation, error) {
	var conv model.Conversation
	err := r.db.
		Preload("Members.User").
		Where("id = ?", id).
		First(&conv).Error
	if err != nil {
		return nil, err
	}
	return &conv, nil
}

// FindPrivateConversation finds an existing private conversation between two users
func (r *ConversationRepository) FindPrivateConversation(userID1, userID2 uuid.UUID) (*model.Conversation, error) {
	var conv model.Conversation
	err := r.db.
		Table("conversations").
		Joins("JOIN conversation_members cm1 ON cm1.conversation_id = conversations.id").
		Joins("JOIN conversation_members cm2 ON cm2.conversation_id = conversations.id").
		Where("conversations.type = ?", model.ConversationTypePrivate).
		Where("cm1.user_id = ?", userID1).
		Where("cm2.user_id = ?", userID2).
		Preload("Members.User").
		First(&conv).Error
	if err != nil {
		return nil, err
	}
	return &conv, nil
}

// GetUserConversations returns all conversations for a user, ordered by latest activity
func (r *ConversationRepository) GetUserConversations(userID uuid.UUID) ([]model.Conversation, error) {
	var conversations []model.Conversation
	err := r.db.
		Joins("JOIN conversation_members ON conversation_members.conversation_id = conversations.id").
		Where("conversation_members.user_id = ? AND conversation_members.deleted_at IS NULL", userID).
		Preload("Members.User").
		Order("conversations.updated_at DESC").
		Find(&conversations).Error
	return conversations, err
}

// AddMember adds a user to a conversation
func (r *ConversationRepository) AddMember(member *model.ConversationMember) error {
	return r.db.Create(member).Error
}

// RemoveMember soft-deletes a member from a conversation
func (r *ConversationRepository) RemoveMember(conversationID, userID uuid.UUID) error {
	return r.db.
		Where("conversation_id = ? AND user_id = ?", conversationID, userID).
		Delete(&model.ConversationMember{}).Error
}

// IsMember checks if a user is a member of a conversation
func (r *ConversationRepository) IsMember(conversationID, userID uuid.UUID) (bool, error) {
	var count int64
	err := r.db.Model(&model.ConversationMember{}).
		Where("conversation_id = ? AND user_id = ?", conversationID, userID).
		Count(&count).Error
	return count > 0, err
}

// GetMemberIDs returns all member user IDs for a conversation
func (r *ConversationRepository) GetMemberIDs(conversationID uuid.UUID) ([]uuid.UUID, error) {
	var memberIDs []uuid.UUID
	err := r.db.Model(&model.ConversationMember{}).
		Where("conversation_id = ?", conversationID).
		Pluck("user_id", &memberIDs).Error
	return memberIDs, err
}

// TouchUpdatedAt bumps the updated_at timestamp (to sort by latest activity)
func (r *ConversationRepository) TouchUpdatedAt(conversationID uuid.UUID) error {
	return r.db.Model(&model.Conversation{}).
		Where("id = ?", conversationID).
		Update("updated_at", gorm.Expr("NOW()")).Error
}

// UpdateLastRead updates the last_read_at timestamp for a member
func (r *ConversationRepository) UpdateLastRead(conversationID, userID uuid.UUID) error {
	return r.db.Model(&model.ConversationMember{}).
		Where("conversation_id = ? AND user_id = ?", conversationID, userID).
		Update("last_read_at", gorm.Expr("NOW()")).Error
}
