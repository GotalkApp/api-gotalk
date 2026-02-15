package handler

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/quocanhngo/gotalk/internal/model"
	"github.com/quocanhngo/gotalk/internal/service"
	"github.com/quocanhngo/gotalk/internal/ws"
	"github.com/quocanhngo/gotalk/pkg/auth"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // In production, validate origin
	},
}

// WSHandler handles WebSocket connections
type WSHandler struct {
	hub         *ws.Hub
	chatService *service.ChatService
	jwtManager  *auth.JWTManager
}

func NewWSHandler(hub *ws.Hub, chatService *service.ChatService, jwtManager *auth.JWTManager) *WSHandler {
	return &WSHandler{
		hub:         hub,
		chatService: chatService,
		jwtManager:  jwtManager,
	}
}

// HandleWebSocket upgrades HTTP to WebSocket and manages the connection
// Client connects with: ws://host/ws?token=<jwt_token>
func (h *WSHandler) HandleWebSocket(c *gin.Context) {
	// Authenticate via query parameter (WebSocket can't use Authorization header)
	tokenString := c.Query("token")
	if tokenString == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Token required"})
		return
	}

	claims, err := h.jwtManager.ValidateToken(tokenString)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		return
	}

	// Upgrade HTTP to WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	// Create client and register with hub
	// Use Name from claims
	client := ws.NewClient(h.hub, conn, claims.UserID, claims.Name)
	h.hub.Register(client)

	log.Printf("✅ WS Connected: UserID=%s Name=%s", claims.UserID, claims.Name)

	// Start read/write pumps in goroutines
	go client.WritePump()
	go client.ReadPump(h.handleWSMessage)
}

// handleWSMessage processes incoming WebSocket messages from clients
func (h *WSHandler) handleWSMessage(client *ws.Client, event model.WSEvent) {
	log.Printf("📩 WS Received from %s (%s): %s", client.Name, client.UserID, event.Type)

	switch event.Type {
	case model.WSEventNewMessage:
		h.handleNewMessage(client, event)

	case model.WSEventTyping:
		h.handleTyping(client, event)

	case model.WSEventStopTyping:
		h.handleStopTyping(client, event)

	case model.WSEventMessageRead:
		h.handleMessageRead(client, event)

	// WebRTC Signaling events
	case model.WSEventCallOffer:
		h.handleCallSignaling(client, event)

	case model.WSEventCallAnswer:
		h.handleCallSignaling(client, event)

	case model.WSEventCallICE:
		h.handleCallSignaling(client, event)

	case model.WSEventCallHangup:
		h.handleCallSignaling(client, event)

	default:
		log.Printf("Unknown WebSocket event type: %s", event.Type)
	}
}

// handleNewMessage processes a new chat message via WebSocket
func (h *WSHandler) handleNewMessage(client *ws.Client, event model.WSEvent) {
	// Parse the payload
	payloadBytes, _ := json.Marshal(event.Payload)
	var payload struct {
		ConversationID uuid.UUID `json:"conversation_id"`
		Content        string    `json:"content"`
		Type           string    `json:"type"`
	}
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		log.Printf("Error parsing new_message payload: %v", err)
		return
	}

	// Save message to DB via service
	msgType := model.MessageType(payload.Type)
	if msgType == "" {
		msgType = model.MessageTypeText
	}

	msg, err := h.chatService.SendMessage(client.UserID, payload.ConversationID, model.SendMessageRequest{
		Content: payload.Content,
		Type:    msgType,
	})
	if err != nil {
		log.Printf("Error saving message: %v", err)
		return
	}

	// Get all members of the conversation
	memberIDs, err := h.chatService.GetConversationMemberIDs(payload.ConversationID)
	if err != nil {
		log.Printf("Error getting member IDs: %v", err)
		return
	}

	// Broadcast new message to all conversation members
	broadcastEvent := &model.WSEvent{
		Type:    model.WSEventNewMessage,
		Payload: msg,
	}
	
	log.Printf("📢 Broadcasting 'new_message' to %d members of conv %s", len(memberIDs), payload.ConversationID)
	h.hub.SendToUsers(memberIDs, broadcastEvent)
}

// handleTyping broadcasts typing indicator to conversation members
func (h *WSHandler) handleTyping(client *ws.Client, event model.WSEvent) {
	payloadBytes, _ := json.Marshal(event.Payload)
	var payload struct {
		ConversationID uuid.UUID `json:"conversation_id"`
	}
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return
	}

	memberIDs, _ := h.chatService.GetConversationMemberIDs(payload.ConversationID)

	typingEvent := &model.WSEvent{
		Type: model.WSEventTyping,
		Payload: model.TypingEvent{
			ConversationID: payload.ConversationID,
			UserID:         client.UserID,
			Name:           client.Name, // Use Name instead of Username
		},
	}

	// Send to all members except the sender
	for _, memberID := range memberIDs {
		if memberID != client.UserID {
			h.hub.SendToUser(memberID, typingEvent)
		}
	}
}

// handleStopTyping broadcasts stop typing indicator
func (h *WSHandler) handleStopTyping(client *ws.Client, event model.WSEvent) {
	payloadBytes, _ := json.Marshal(event.Payload)
	var payload struct {
		ConversationID uuid.UUID `json:"conversation_id"`
	}
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return
	}

	memberIDs, _ := h.chatService.GetConversationMemberIDs(payload.ConversationID)

	stopEvent := &model.WSEvent{
		Type: model.WSEventStopTyping,
		Payload: model.TypingEvent{
			ConversationID: payload.ConversationID,
			UserID:         client.UserID,
			Name:           client.Name, // Use Name instead of Username
		},
	}

	for _, memberID := range memberIDs {
		if memberID != client.UserID {
			h.hub.SendToUser(memberID, stopEvent)
		}
	}
}

// handleMessageRead processes read receipt events
func (h *WSHandler) handleMessageRead(client *ws.Client, event model.WSEvent) {
	payloadBytes, _ := json.Marshal(event.Payload)
	var payload struct {
		ConversationID uuid.UUID `json:"conversation_id"`
		MessageID      uuid.UUID `json:"message_id"`
	}
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return
	}

	// Mark messages as read in DB
	_ = h.chatService.MarkMessagesAsRead(payload.ConversationID, client.UserID)

	// Notify other members about read receipt
	memberIDs, _ := h.chatService.GetConversationMemberIDs(payload.ConversationID)

	readEvent := &model.WSEvent{
		Type: model.WSEventMessageRead,
		Payload: model.MessageReadEvent{
			ConversationID: payload.ConversationID,
			MessageID:      payload.MessageID,
			UserID:         client.UserID,
		},
	}

	for _, memberID := range memberIDs {
		if memberID != client.UserID {
			h.hub.SendToUser(memberID, readEvent)
		}
	}
}

// handleCallSignaling forwards WebRTC signaling events to the target user
func (h *WSHandler) handleCallSignaling(client *ws.Client, event model.WSEvent) {
	payloadBytes, _ := json.Marshal(event.Payload)
	var payload struct {
		To uuid.UUID `json:"to"`
	}
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		log.Printf("Error parsing call signaling payload: %v", err)
		return
	}

	// Forward the event as-is to the target user
	h.hub.SendToUser(payload.To, &event)
}
