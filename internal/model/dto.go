package model

import "github.com/google/uuid"

// ========== Auth DTOs ==========

type RegisterRequest struct {
	Name     string `json:"name" binding:"required,min=2,max=100"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

type GoogleLoginRequest struct {
	IDToken string `json:"id_token" binding:"required"` // Google ID token from frontend
}

type LoginResponse struct {
	Token string       `json:"token"`
	User  UserResponse `json:"user"`
}

// ========== OTP DTOs ==========

type VerifyOTPRequest struct {
	Email string `json:"email" binding:"required,email"`
	Code  string `json:"code" binding:"required,len=6"`
}

type ResendOTPRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type OTPSentResponse struct {
	Message   string `json:"message"`
	Email     string `json:"email"`
	ExpiresIn int    `json:"expires_in"` // seconds until code expires
}

type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type ResetPasswordRequest struct {
	Email       string `json:"email" binding:"required,email"`
	Code        string `json:"code" binding:"required,len=6"`
	NewPassword string `json:"new_password" binding:"required,min=6"`
}

// ========== Google OAuth DTOs ==========

type GoogleUserInfo struct {
	GoogleID string `json:"sub"`
	Email    string `json:"email"`
	Name     string `json:"name"`
	Picture  string `json:"picture"`
	Verified bool   `json:"email_verified"`
}

type UpdateProfileRequest struct {
	Name   string `json:"name" binding:"max=100"`
	Avatar string `json:"avatar" binding:"max=500"`
}

type UpdateSettingsRequest struct {
	Theme                 string `json:"theme" binding:"omitempty,oneof=light dark system"`
	IsNotificationEnabled *bool  `json:"is_notification_enabled"`
	IsSoundEnabled        *bool  `json:"is_sound_enabled"`
	Language              string `json:"language" binding:"omitempty,len=2"`
}

type RegisterDeviceRequest struct {
	FCMToken   string `json:"fcm_token" binding:"required"`
	DeviceType string `json:"device_type" binding:"required"`
}

// ========== Conversation DTOs ==========

type CreateConversationRequest struct {
	Type      ConversationType `json:"type" binding:"required,oneof=private group"`
	Name      string           `json:"name"` // required for group
	MemberIDs []uuid.UUID      `json:"member_ids" binding:"required,min=1"`
}

type DirectConversationRequest struct {
	ReceiverID uuid.UUID `json:"receiver_id" binding:"required"`
}

type DirectConversationResponse struct {
	Conversation ConversationResponse `json:"conversation"`
	Messages     []Message            `json:"messages"`
	IsNew        bool                 `json:"is_new"`
}

type ConversationResponse struct {
	Conversation
	UnreadCount int `json:"unread_count"`
}

// ========== Message DTOs ==========

type SendMessageRequest struct {
	Content     string            `json:"content" binding:"required_without_all=Attachments FileURL"`
	Type        MessageType       `json:"type"`
	ReplyToID   *uuid.UUID        `json:"reply_to_id"`
	Attachments []AttachmentInput `json:"attachments,omitempty"`
	// Legacy single-file fields (backward compatible)
	FileURL  string `json:"file_url,omitempty"`
	FileName string `json:"file_name,omitempty"`
	FileSize int64  `json:"file_size,omitempty"`
}

// AttachmentInput is used when sending a message with attachments
type AttachmentInput struct {
	URL      string         `json:"url" binding:"required"`
	Type     AttachmentType `json:"type" binding:"required"`
	FileName string         `json:"file_name"`
	FileSize int64          `json:"file_size"`
	MimeType string         `json:"mime_type"`
}

type MessageListRequest struct {
	Before string `form:"before"` // cursor for pagination (message ID)
	Limit  int    `form:"limit,default=50"`
}

// ========== WebSocket Event DTOs ==========

type WSEvent struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

// WebSocket event types
const (
	WSEventNewMessage  = "new_message"
	WSEventTyping      = "typing"
	WSEventStopTyping  = "stop_typing"
	WSEventOnline      = "online"
	WSEventOffline     = "offline"
	WSEventMessageRead = "message_read"
	WSEventCallOffer   = "call_offer"
	WSEventCallAnswer  = "call_answer"
	WSEventCallICE     = "call_ice_candidate"
	WSEventCallHangup  = "call_hangup"
)

type TypingEvent struct {
	ConversationID uuid.UUID `json:"conversation_id"`
	UserID         uuid.UUID `json:"user_id"`
	Name           string    `json:"name"`
}

type OnlineEvent struct {
	UserID   uuid.UUID `json:"user_id"`
	IsOnline bool      `json:"is_online"`
}

type MessageReadEvent struct {
	ConversationID uuid.UUID `json:"conversation_id"`
	MessageID      uuid.UUID `json:"message_id"`
	UserID         uuid.UUID `json:"user_id"`
}

// ========== WebRTC Signaling DTOs ==========

type CallOfferEvent struct {
	From           uuid.UUID   `json:"from"`
	To             uuid.UUID   `json:"to"`
	ConversationID uuid.UUID   `json:"conversation_id"`
	SDP            interface{} `json:"sdp"`
	CallType       string      `json:"call_type"` // "audio" or "video"
}

type CallAnswerEvent struct {
	From           uuid.UUID   `json:"from"`
	To             uuid.UUID   `json:"to"`
	ConversationID uuid.UUID   `json:"conversation_id"`
	SDP            interface{} `json:"sdp"`
}

type ICECandidateEvent struct {
	From           uuid.UUID   `json:"from"`
	To             uuid.UUID   `json:"to"`
	ConversationID uuid.UUID   `json:"conversation_id"`
	Candidate      interface{} `json:"candidate"`
}

// ========== Common ==========

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

type SuccessResponse struct {
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}
