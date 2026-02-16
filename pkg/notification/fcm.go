package notification

import (
	"context"
	"fmt"
	"log"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"github.com/google/uuid"
	"github.com/quocanhngo/gotalk/internal/repository"
	"google.golang.org/api/option"
)

// NotificationService handles FCM notifications
type NotificationService struct {
	client   *messaging.Client
	userRepo *repository.UserRepository
}

// NewNotificationService creates a new FCM notification service
func NewNotificationService(credentialsFile string, userRepo *repository.UserRepository) (*NotificationService, error) {
	if credentialsFile == "" {
		log.Println("⚠️ Firebase credentials not provided, push notifications disabled")
		return nil, nil
	}

	opt := option.WithCredentialsFile(credentialsFile)
	app, err := firebase.NewApp(context.Background(), nil, opt)
	if err != nil {
		// Log warning instead of error to not block server startup
		log.Printf("⚠️ Failed to initialize Firebase app: %v (push notifications disabled)", err)
		return nil, nil
	}

	client, err := app.Messaging(context.Background())
	if err != nil {
		log.Printf("⚠️ Failed to get messaging client: %v", err)
		return nil, nil
	}

	log.Println("✅ Firebase FCM initialized")
	return &NotificationService{
		client:   client,
		userRepo: userRepo,
	}, nil
}

// SendMessageNotification sends a push notification for a new chat message
func (s *NotificationService) SendMessageNotification(ctx context.Context, receiverID uuid.UUID, senderName string, content string, conversationID uuid.UUID) error {
	if s == nil || s.client == nil {
		return nil
	}

	// Check if user has notifications enabled
	user, err := s.userRepo.FindByID(receiverID)
	if err != nil {
		return err
	}
	if !user.IsNotificationEnabled {
		return nil
	}

	// Get user devices
	devices, err := s.userRepo.GetUserDevices(receiverID)
	if err != nil {
		return err
	}

	if len(devices) == 0 {
		return nil
	}

	if content == "" {
		content = "Sent an attachment"
	}

	// Prepare token list
	tokens := make([]string, 0, len(devices))
	for _, d := range devices {
		tokens = append(tokens, d.FCMToken)
	}

	// Create message
	message := &messaging.MulticastMessage{
		Tokens: tokens,
		Notification: &messaging.Notification{
			Title: senderName,
			Body:  content,
		},
		Data: map[string]string{
			"type":            "new_message",
			"conversation_id": conversationID.String(),
			"sender_name":     senderName,
		},
		Android: &messaging.AndroidConfig{
			Priority: "high",
			Notification: &messaging.AndroidNotification{
				ClickAction: "FLUTTER_NOTIFICATION_CLICK",
			},
		},
		APNS: &messaging.APNSConfig{
			Payload: &messaging.APNSPayload{
				Aps: &messaging.Aps{
					Sound: "default",
				},
			},
		},
	}

	// Send
	br, err := s.client.SendMulticast(ctx, message)
	if err != nil {
		return fmt.Errorf("error sending multicast message: %w", err)
	}

	if br.FailureCount > 0 {
		// Log failures
		for idx, resp := range br.Responses {
			if !resp.Success {
				log.Printf("⚠️ FCM failure for token %s: %v", tokens[idx], resp.Error)
			}
		}
	}

	return nil
}
