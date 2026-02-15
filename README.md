# 🚀 GoTalk - Real-time Chat & Video Call API

A high-performance, scalable backend for real-time messaging and video calling, built with Go (Gin), WebSockets, Redis Pub/Sub, and PostgreSQL.

## 🏗️ Architecture

```
┌─────────────┐     ┌──────────────┐     ┌─────────────┐
│   Frontend   │────▶│   Traefik    │────▶│  GoTalk API │
│  (React/Next)│     │ (L7 Proxy)   │     │  (Go + Gin) │
└─────────────┘     └──────────────┘     └──────┬──────┘
                                                 │
                    ┌────────────────────────────┤
                    │              │              │
              ┌─────▼─────┐ ┌─────▼─────┐ ┌─────▼─────┐
              │ PostgreSQL │ │   Redis   │ │   MinIO   │
              │ (Messages) │ │ (Pub/Sub) │ │  (Files)  │
              └───────────┘ └───────────┘ └───────────┘
```

## 📁 Project Structure

```
chat-api/
├── cmd/
│   └── server/
│       └── main.go              # Entry point
├── internal/
│   ├── config/
│   │   └── config.go            # Configuration management
│   ├── handler/
│   │   ├── auth_handler.go      # Auth endpoints
│   │   ├── chat_handler.go      # Chat REST endpoints
│   │   └── ws_handler.go        # WebSocket handler
│   ├── middleware/
│   │   ├── auth.go              # JWT auth middleware
│   │   └── cors.go              # CORS middleware
│   ├── model/
│   │   ├── user.go              # User model
│   │   ├── message.go           # Message model
│   │   ├── conversation.go      # Conversation model
│   │   └── dto.go               # Request/Response DTOs
│   ├── repository/
│   │   ├── user_repo.go         # User data access
│   │   ├── conversation_repo.go # Conversation data access
│   │   └── message_repo.go      # Message data access
│   ├── service/
│   │   ├── auth_service.go      # Auth business logic
│   │   └── chat_service.go      # Chat business logic
│   └── ws/
│       ├── hub.go               # WebSocket hub + Redis Pub/Sub
│       └── client.go            # WebSocket client connection
├── pkg/
│   └── auth/
│       └── jwt.go               # JWT token manager
├── docker-compose.yml           # Development stack
├── Dockerfile                   # Multi-stage build
├── .air.toml                    # Hot reload config
├── .env                         # Environment variables
└── go.mod
```

## 🚀 Quick Start

### Prerequisites
- Docker & Docker Compose
- Go 1.23+ (for local development)

### 1. Start with Docker Compose

```bash
# Start all services (Traefik, API, PostgreSQL, Redis, MinIO)
docker compose up -d

# View logs
docker compose logs -f api
```

### 2. Access Points

| Service       | URL                           |
|---------------|-------------------------------|
| API           | http://api.localhost          |
| Traefik Dash  | http://localhost:8090         |
| MinIO Console | http://localhost:9001         |
| PostgreSQL    | localhost:5432               |
| Redis         | localhost:6379               |

### 3. Health Check

```bash
curl http://api.localhost/health
```

## 📡 API Endpoints

### Auth
```
POST /api/v1/auth/register       # Register new user
POST /api/v1/auth/login          # Login
GET  /api/v1/auth/profile        # Get profile (auth required)
```

### Users
```
GET  /api/v1/users/search?q=     # Search users (auth required)
```

### Conversations
```
GET  /api/v1/conversations       # List conversations
POST /api/v1/conversations       # Create conversation
GET  /api/v1/conversations/:id   # Get conversation details
```

### Messages
```
GET  /api/v1/conversations/:id/messages   # Get messages (paginated)
POST /api/v1/conversations/:id/messages   # Send message
POST /api/v1/conversations/:id/read       # Mark as read
```

### WebSocket
```
GET  /ws?token=<jwt_token>       # Connect WebSocket
```

## 🔌 WebSocket Events

### Client → Server
```json
// Send message
{"type": "new_message", "payload": {"conversation_id": "uuid", "content": "Hello!"}}

// Typing indicator
{"type": "typing", "payload": {"conversation_id": "uuid"}}

// Stop typing
{"type": "stop_typing", "payload": {"conversation_id": "uuid"}}

// Read receipt
{"type": "message_read", "payload": {"conversation_id": "uuid", "message_id": "uuid"}}

// WebRTC Call Offer
{"type": "call_offer", "payload": {"to": "user_uuid", "sdp": {...}, "call_type": "video"}}

// WebRTC Call Answer
{"type": "call_answer", "payload": {"to": "user_uuid", "sdp": {...}}}

// WebRTC ICE Candidate
{"type": "call_ice_candidate", "payload": {"to": "user_uuid", "candidate": {...}}}
```

### Server → Client
```json
// New message received
{"type": "new_message", "payload": {/* message object */}}

// User typing
{"type": "typing", "payload": {"conversation_id": "uuid", "user_id": "uuid", "username": "john"}}

// User online/offline
{"type": "online", "payload": {"user_id": "uuid", "is_online": true}}
```

## 🔧 Frontend Integration

### Connect from Frontend docker-compose.yml

```yaml
# In your frontend's docker-compose.yml
services:
  frontend:
    # ... your frontend config
    networks:
      - gotalk-network
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.frontend.rule=Host(`chat.localhost`)"
      - "traefik.http.routers.frontend.entrypoints=web"
      - "traefik.http.services.frontend.loadbalancer.server.port=3000"

networks:
  gotalk-network:
    external: true
```

## 🏋️ Key Technical Highlights (CV Points)

1. **Distributed WebSocket with Redis Pub/Sub** - Horizontal scaling support
2. **Cursor-based Pagination** - Efficient message loading
3. **Multi-device Support** - One user, multiple connections
4. **WebRTC Signaling Server** - Custom signaling for video calls
5. **Graceful Shutdown** - Clean connection handling
6. **JWT Authentication** - Stateless auth with bcrypt
7. **Docker + Traefik** - Production-ready infrastructure

## 📜 License

MIT
