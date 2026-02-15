package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/quocanhngo/gotalk/internal/model"
	"github.com/quocanhngo/gotalk/internal/service"
	"github.com/quocanhngo/gotalk/internal/ws"
)

// ChatHandler handles chat-related HTTP endpoints
type ChatHandler struct {
	chatService *service.ChatService
	hub         *ws.Hub
}

func NewChatHandler(chatService *service.ChatService, hub *ws.Hub) *ChatHandler {
	return &ChatHandler{chatService: chatService, hub: hub}
}

// GetOrCreateDirect godoc
// @Summary Get or create direct conversation
// @Description Find existing private chat with user, or create new one. Returns conversation + messages.
// @Tags Conversations
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body model.DirectConversationRequest true "Partner ID"
// @Success 200 {object} model.DirectConversationResponse
// @Failure 400 {object} model.ErrorResponse
// @Router /conversations/direct [post]
func (h *ChatHandler) GetOrCreateDirect(c *gin.Context) {
	var req model.DirectConversationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Error: "Invalid request", Message: err.Error()})
		return
	}

	userID := c.MustGet("user_id").(uuid.UUID)
	resp, err := h.chatService.GetOrCreateDirect(userID, req.ReceiverID)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// CreateConversation godoc
// @Summary Create a new conversation
// @Tags Chat
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body model.CreateConversationRequest true "Create conversation request"
// @Success 201 {object} model.Conversation
// @Router /conversations [post]
func (h *ChatHandler) CreateConversation(c *gin.Context) {
	var req model.CreateConversationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Error: "Invalid request", Message: err.Error()})
		return
	}

	userID := c.MustGet("user_id").(uuid.UUID)
	conv, err := h.chatService.CreateConversation(userID, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusCreated, conv)
}

// GetConversations godoc
// @Summary Get all conversations for the current user
// @Tags Chat
// @Produce json
// @Security BearerAuth
// @Success 200 {array} model.ConversationResponse
// @Router /conversations [get]
func (h *ChatHandler) GetConversations(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	conversations, err := h.chatService.GetConversations(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Error: "Failed to get conversations"})
		return
	}

	c.JSON(http.StatusOK, conversations)
}

// GetConversation godoc
// @Summary Get a specific conversation
// @Tags Chat
// @Produce json
// @Security BearerAuth
// @Param id path string true "Conversation ID"
// @Success 200 {object} model.Conversation
// @Router /conversations/{id} [get]
func (h *ChatHandler) GetConversation(c *gin.Context) {
	convID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Error: "Invalid conversation ID"})
		return
	}

	userID := c.MustGet("user_id").(uuid.UUID)
	conv, err := h.chatService.GetConversation(convID, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, model.ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, conv)
}

// SendMessage godoc
// @Summary Send a message to a conversation
// @Tags Chat
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Conversation ID"
// @Param body body model.SendMessageRequest true "Send message request"
// @Success 201 {object} model.Message
// @Router /conversations/{id}/messages [post]
func (h *ChatHandler) SendMessage(c *gin.Context) {
	convID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Error: "Invalid conversation ID"})
		return
	}

	var req model.SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Error: "Invalid request", Message: err.Error()})
		return
	}

	userID := c.MustGet("user_id").(uuid.UUID)
	msg, err := h.chatService.SendMessage(userID, convID, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Error: err.Error()})
		return
	}

	// Broadcast to recipients via WebSocket (send only to others)
	go func() {
		memberIDs, err := h.chatService.GetConversationMemberIDs(convID)
		if err == nil {
			var recipientIDs []uuid.UUID
			for _, id := range memberIDs {
				if id != userID {
					recipientIDs = append(recipientIDs, id)
				}
			}

			if len(recipientIDs) > 0 {
				broadcastEvent := &model.WSEvent{
					Type:    model.WSEventNewMessage,
					Payload: msg,
				}
				h.hub.SendToUsers(recipientIDs, broadcastEvent)
			}
		}
	}()

	c.JSON(http.StatusCreated, msg)
}

// GetMessages godoc
// @Summary Get messages for a conversation
// @Tags Chat
// @Produce json
// @Security BearerAuth
// @Param id path string true "Conversation ID"
// @Param before query string false "Cursor: message ID to get messages before"
// @Param limit query int false "Number of messages to return (default: 50)"
// @Success 200 {array} model.Message
// @Router /conversations/{id}/messages [get]
func (h *ChatHandler) GetMessages(c *gin.Context) {
	convID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Error: "Invalid conversation ID"})
		return
	}

	var req model.MessageListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Error: "Invalid request"})
		return
	}

	var before *uuid.UUID
	if req.Before != "" {
		parsed, err := uuid.Parse(req.Before)
		if err == nil {
			before = &parsed
		}
	}

	userID := c.MustGet("user_id").(uuid.UUID)
	messages, err := h.chatService.GetMessages(convID, userID, before, req.Limit)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, messages)
}

// MarkAsRead godoc
// @Summary Mark all messages in a conversation as read
// @Tags Chat
// @Produce json
// @Security BearerAuth
// @Param id path string true "Conversation ID"
// @Success 200 {object} model.SuccessResponse
// @Router /conversations/{id}/read [post]
func (h *ChatHandler) MarkAsRead(c *gin.Context) {
	convID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Error: "Invalid conversation ID"})
		return
	}

	userID := c.MustGet("user_id").(uuid.UUID)
	if err := h.chatService.MarkMessagesAsRead(convID, userID); err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Error: "Failed to mark as read"})
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse{Message: "Messages marked as read"})
}
