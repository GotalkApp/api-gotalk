package service

import (
	"errors"

	"github.com/google/uuid"
	"github.com/quocanhngo/gotalk/internal/model"
	"github.com/quocanhngo/gotalk/internal/repository"
	"gorm.io/gorm"
)

// ChatService handles chat business logic
type ChatService struct {
	convRepo *repository.ConversationRepository
	msgRepo  *repository.MessageRepository
	userRepo *repository.UserRepository
}

func NewChatService(
	convRepo *repository.ConversationRepository,
	msgRepo *repository.MessageRepository,
	userRepo *repository.UserRepository,
) *ChatService {
	return &ChatService{
		convRepo: convRepo,
		msgRepo:  msgRepo,
		userRepo: userRepo,
	}
}

// CreateConversation creates a new conversation (private or group)
func (s *ChatService) CreateConversation(creatorID uuid.UUID, req model.CreateConversationRequest) (*model.Conversation, error) {
	// For private conversations, check if one already exists
	if req.Type == model.ConversationTypePrivate {
		if len(req.MemberIDs) != 1 {
			return nil, errors.New("private conversation requires exactly 1 other member")
		}

		existingConv, err := s.convRepo.FindPrivateConversation(creatorID, req.MemberIDs[0])
		if err == nil {
			return existingConv, nil // Return existing conversation
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	}

	// Create conversation
	conv := &model.Conversation{
		Name:      req.Name,
		Type:      req.Type,
		CreatorID: &creatorID,
	}

	// Add creator as admin
	members := []model.ConversationMember{
		{
			UserID: creatorID,
			Role:   model.MemberRoleAdmin,
		},
	}

	// Add other members
	for _, memberID := range req.MemberIDs {
		if memberID == creatorID {
			continue // Skip if creator is in the list
		}
		members = append(members, model.ConversationMember{
			UserID: memberID,
			Role:   model.MemberRoleMember,
		})
	}

	conv.Members = members

	if err := s.convRepo.Create(conv); err != nil {
		return nil, errors.New("failed to create conversation")
	}

	// Reload with relations
	return s.convRepo.FindByID(conv.ID)
}

// GetOrCreateDirect finds or creates a private conversation
func (s *ChatService) GetOrCreateDirect(myID, partnerID uuid.UUID) (*model.DirectConversationResponse, error) {
	// 1. Try to find existing private conv
	conv, err := s.convRepo.FindPrivateConversation(myID, partnerID)
	if err == nil {
		// Found! Mark as read immediately
		_ = s.convRepo.UpdateLastRead(conv.ID, myID)

		// Get messages
		msgs, _ := s.msgRepo.GetConversationMessages(conv.ID, nil, 50)

		// Count unread
		unreadCount, _ := s.msgRepo.CountUnread(conv.ID, myID)

		// Get last message
		lastMsg, _ := s.msgRepo.GetLastMessage(conv.ID)

		// Populate name/avatar for private chat
		if conv.Type == model.ConversationTypePrivate {
			for _, m := range conv.Members {
				if m.UserID != myID {
					conv.Name = m.User.Name
					conv.Avatar = m.User.Avatar
					break
				}
			}
		}

		// Build response
		convResp := model.ConversationResponse{
			Conversation: *conv,
			UnreadCount:  int(unreadCount),
		}
		convResp.LastMessage = lastMsg

		return &model.DirectConversationResponse{
			Conversation: convResp,
			Messages:     msgs,
			IsNew:        false,
		}, nil
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	// 2. Not found -> Create new
	// We need partner info to set conversation name (optional, usually private chat name is dynamic)
	newConv, err := s.CreateConversation(myID, model.CreateConversationRequest{
		Type:      model.ConversationTypePrivate,
		MemberIDs: []uuid.UUID{partnerID},
	})
	if err != nil {
		return nil, err
	}

	return &model.DirectConversationResponse{
		Conversation: model.ConversationResponse{
			Conversation: *newConv,
			UnreadCount:  0,
		},
		Messages: []model.Message{},
		IsNew:    true,
	}, nil
}

// GetConversations returns all conversations for a user
func (s *ChatService) GetConversations(userID uuid.UUID) ([]model.ConversationResponse, error) {
	conversations, err := s.convRepo.GetUserConversations(userID)
	if err != nil {
		return nil, err
	}

	result := []model.ConversationResponse{}
	for i := range conversations {
		// Get last message for each conversation
		lastMsg, _ := s.msgRepo.GetLastMessage(conversations[i].ID)
		conversations[i].LastMessage = lastMsg

		// Count unread messages
		unreadCount, _ := s.msgRepo.CountUnread(conversations[i].ID, userID)

		// Populate name/avatar for private chat
		conv := conversations[i]
		if conv.Type == model.ConversationTypePrivate {
			for _, m := range conv.Members {
				if m.UserID != userID {
					conv.Name = m.User.Name
					conv.Avatar = m.User.Avatar
					break
				}
			}
		}

		result = append(result, model.ConversationResponse{
			Conversation: conv,
			UnreadCount:  int(unreadCount),
		})
	}

	return result, nil
}

// GetConversation returns a specific conversation
func (s *ChatService) GetConversation(convID, userID uuid.UUID) (*model.Conversation, error) {
	// Check membership
	isMember, err := s.convRepo.IsMember(convID, userID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, errors.New("you are not a member of this conversation")
	}

	return s.convRepo.FindByID(convID)
}

// SendMessage sends a message to a conversation
func (s *ChatService) SendMessage(senderID, convID uuid.UUID, req model.SendMessageRequest) (*model.Message, error) {
	// Check membership
	isMember, err := s.convRepo.IsMember(convID, senderID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, errors.New("you are not a member of this conversation")
	}

	msgType := req.Type
	if msgType == "" {
		msgType = model.MessageTypeText
		// Auto-detect type from attachments
		if len(req.Attachments) > 0 {
			msgType = model.MessageType(req.Attachments[0].Type)
		} else if req.FileURL != "" {
			msgType = model.MessageTypeFile
		}
	}

	msg := &model.Message{
		ConversationID: convID,
		SenderID:       senderID,
		Content:        req.Content,
		Type:           msgType,
		Status:         model.MessageStatusSent,
		FileURL:        req.FileURL,
		FileName:       req.FileName,
		FileSize:       req.FileSize,
		ReplyToID:      req.ReplyToID,
	}

	if err := s.msgRepo.Create(msg); err != nil {
		return nil, errors.New("failed to send message")
	}

	// Save attachments if any
	if len(req.Attachments) > 0 {
		for _, att := range req.Attachments {
			attachment := model.MessageAttachment{
				MessageID: msg.ID,
				Type:      att.Type,
				URL:       att.URL,
				FileName:  att.FileName,
				FileSize:  att.FileSize,
				MimeType:  att.MimeType,
			}
			s.msgRepo.CreateAttachment(&attachment)
		}
	}

	// Update conversation's updated_at for sorting
	_ = s.convRepo.TouchUpdatedAt(convID)

	// Reload with sender info and attachments
	return s.msgRepo.FindByID(msg.ID)
}

// GetMessages returns paginated messages for a conversation
func (s *ChatService) GetMessages(convID, userID uuid.UUID, before *uuid.UUID, limit int) ([]model.Message, error) {
	// Check membership
	isMember, err := s.convRepo.IsMember(convID, userID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, errors.New("you are not a member of this conversation")
	}

	if limit <= 0 || limit > 100 {
		limit = 50
	}

	return s.msgRepo.GetConversationMessages(convID, before, limit)
}

// MarkMessagesAsRead updates the last_read_at timestamp
func (s *ChatService) MarkMessagesAsRead(convID, userID uuid.UUID) error {
	return s.convRepo.UpdateLastRead(convID, userID)
}

// GetConversationMemberIDs returns all member IDs for a conversation
func (s *ChatService) GetConversationMemberIDs(convID uuid.UUID) ([]uuid.UUID, error) {
	return s.convRepo.GetMemberIDs(convID)
}
