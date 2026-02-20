# ============================================
# GoTalk - Dockerfile
# Multi-stage build for Development & Production
# ============================================

# Stage 1: Development (Hot Reload with Air)
# Use 'alpine' tag for latest stable Go version (currently 1.22+)
FROM golang:alpine AS development

# Install git required for fetching dependencies and tzdata for timezones
RUN apk add --no-cache git tzdata

WORKDIR /app

# Install Air for hot reloading
# Pin version for stability (v1.61.0 is stable)
RUN go install github.com/air-verse/air@v1.61.0

# Download dependencies first (better caching)
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Run Air (configuration in .air.toml)
CMD ["air", "-c", ".air.toml"]


# Stage 2: Builder (Compile optimized binary)
FROM golang:alpine AS builder

WORKDIR /app

# Install dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build binary
# CGO_ENABLED=0: Statically linked binary (important for scratch/alpine)
# -ldflags="-w -s": Strip debug symbols for smaller size
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o main ./cmd/server/main.go


# Stage 3: Production (Minimal Image)
FROM alpine:latest AS production

WORKDIR /app

# Install ca-certificates and tzdata for HTTPS calls and timezones
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user for security
RUN addgroup -S gotalk && adduser -S gotalk -G gotalk
USER gotalk

# Copy binary from builder
COPY --from=builder /app/main .
# Copy migration files (if using file-based migration inside binary, this is optional, 
# but if loading from disk, we need them. Here we use embed, so handled in binary)

# Verify binary execution permission
# (Already executable from build step, but good practice)

# Expose API port
EXPOSE 8080

# Run application
CMD ["./main"]
