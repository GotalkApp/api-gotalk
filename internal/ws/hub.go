package ws

import (
	"context"
	"encoding/json"
	"log"
	"sync"

	"github.com/google/uuid"
	"github.com/quocanhngo/gotalk/internal/model"
	"github.com/redis/go-redis/v9"
)

const redisChannel = "gotalk:messages"

// Hub manages all WebSocket connections and message broadcasting
// It uses Redis Pub/Sub for horizontal scaling across multiple instances
type Hub struct {
	// Map of userID -> set of client connections (one user can have multiple tabs/devices)
	clients    map[uuid.UUID]map[*Client]bool
	mu         sync.RWMutex

	// Channels for registering/unregistering clients
	register   chan *Client
	unregister chan *Client

	// Channel for broadcasting messages to local clients
	broadcast  chan *model.WSEvent

	// Redis client for Pub/Sub (horizontal scaling)
	rdb        *redis.Client

	// Callback when user comes online/offline
	onStatusChange func(userID uuid.UUID, online bool)
}

// NewHub creates a new WebSocket Hub
func NewHub(rdb *redis.Client, onStatusChange func(userID uuid.UUID, online bool)) *Hub {
	return &Hub{
		clients:        make(map[uuid.UUID]map[*Client]bool),
		register:       make(chan *Client),
		unregister:     make(chan *Client),
		broadcast:      make(chan *model.WSEvent, 256),
		rdb:            rdb,
		onStatusChange: onStatusChange,
	}
}

// Run starts the Hub's main event loop
func (h *Hub) Run(ctx context.Context) {
	// Start Redis subscriber in a goroutine
	go h.subscribeRedis(ctx)

	for {
		select {
		case <-ctx.Done():
			return

		case client := <-h.register:
			h.addClient(client)

		case client := <-h.unregister:
			h.removeClient(client)

		case event := <-h.broadcast:
			// This handles local broadcast only
			// For cross-instance, we publish to Redis
			h.broadcastToLocal(event)
		}
	}
}

// Register queues a client for registration with the hub
func (h *Hub) Register(client *Client) {
	h.register <- client
}

// addClient registers a new client connection
func (h *Hub) addClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.clients[client.UserID]; !ok {
		h.clients[client.UserID] = make(map[*Client]bool)
		// User just came online (first connection)
		if h.onStatusChange != nil {
			go h.onStatusChange(client.UserID, true)
		}
		// Broadcast online event
		h.publishToRedis(&model.WSEvent{
			Type: model.WSEventOnline,
			Payload: model.OnlineEvent{
				UserID:   client.UserID,
				IsOnline: true,
			},
		})
	}
	h.clients[client.UserID][client] = true
	log.Printf("✅ Client connected: %s (total connections: %d)", client.UserID, len(h.clients[client.UserID]))
}

// removeClient unregisters a client connection
func (h *Hub) removeClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if clients, ok := h.clients[client.UserID]; ok {
		delete(clients, client)
		close(client.send)

		if len(clients) == 0 {
			// User has no more connections (offline)
			delete(h.clients, client.UserID)
			if h.onStatusChange != nil {
				go h.onStatusChange(client.UserID, false)
			}
			// Broadcast offline event
			h.publishToRedis(&model.WSEvent{
				Type: model.WSEventOffline,
				Payload: model.OnlineEvent{
					UserID:   client.UserID,
					IsOnline: false,
				},
			})
		}
	}
	log.Printf("❌ Client disconnected: %s", client.UserID)
}

// SendToUser sends an event to a specific user (all their connections)
func (h *Hub) SendToUser(userID uuid.UUID, event *model.WSEvent) {
	// Publish to Redis so all instances can deliver
	h.publishToRedis(&TargetedEvent{
		TargetUserID: userID,
		Event:        event,
	})
}

// SendToUsers sends an event to multiple users
func (h *Hub) SendToUsers(userIDs []uuid.UUID, event *model.WSEvent) {
	for _, userID := range userIDs {
		h.SendToUser(userID, event)
	}
}

// sendToLocalUser sends an event to a user on this instance only
func (h *Hub) sendToLocalUser(userID uuid.UUID, event *model.WSEvent) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if clients, ok := h.clients[userID]; ok {
		data, err := json.Marshal(event)
		if err != nil {
			log.Printf("Error marshaling event: %v", err)
			return
		}
		for client := range clients {
			select {
			case client.send <- data:
			default:
				// Client's send buffer is full, close connection
				close(client.send)
				delete(clients, client)
			}
		}
	}
}

// broadcastToLocal sends an event to all connected local clients
func (h *Hub) broadcastToLocal(event *model.WSEvent) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	data, err := json.Marshal(event)
	if err != nil {
		log.Printf("Error marshaling broadcast event: %v", err)
		return
	}

	for _, clients := range h.clients {
		for client := range clients {
			select {
			case client.send <- data:
			default:
				close(client.send)
				delete(clients, client)
			}
		}
	}
}

// IsUserOnline checks if a user has any active connections on this instance
func (h *Hub) IsUserOnline(userID uuid.UUID) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	_, ok := h.clients[userID]
	return ok
}

// GetOnlineUserIDs returns all currently connected user IDs on this instance
func (h *Hub) GetOnlineUserIDs() []uuid.UUID {
	h.mu.RLock()
	defer h.mu.RUnlock()

	userIDs := make([]uuid.UUID, 0, len(h.clients))
	for userID := range h.clients {
		userIDs = append(userIDs, userID)
	}
	return userIDs
}

// ========== Redis Pub/Sub for Horizontal Scaling ==========

// TargetedEvent wraps an event with a target user ID for Redis Pub/Sub
type TargetedEvent struct {
	TargetUserID uuid.UUID      `json:"target_user_id,omitempty"`
	Event        *model.WSEvent `json:"event"`
}

// publishToRedis publishes an event to Redis for cross-instance communication
func (h *Hub) publishToRedis(data interface{}) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Printf("Error marshaling for Redis: %v", err)
		return
	}

	if err := h.rdb.Publish(context.Background(), redisChannel, jsonData).Err(); err != nil {
		log.Printf("Error publishing to Redis: %v", err)
	}
}

// subscribeRedis subscribes to Redis and delivers events to local clients
func (h *Hub) subscribeRedis(ctx context.Context) {
	pubsub := h.rdb.Subscribe(ctx, redisChannel)
	defer pubsub.Close()

	ch := pubsub.Channel()
	log.Println("📡 Redis Pub/Sub subscriber started")

	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-ch:
			var targeted TargetedEvent
			if err := json.Unmarshal([]byte(msg.Payload), &targeted); err != nil {
				log.Printf("Error unmarshaling Redis message: %v", err)
				continue
			}

			if targeted.TargetUserID != uuid.Nil {
				// Targeted event - send to specific user
				h.sendToLocalUser(targeted.TargetUserID, targeted.Event)
			} else if targeted.Event != nil {
				// Broadcast event - send to all local clients
				h.broadcastToLocal(targeted.Event)
			}
		}
	}
}
