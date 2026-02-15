-- ============================================
-- GoTalk: Initial Database Schema
-- Migration: 000001_init_schema
-- ============================================

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ==================== ENUM Types ====================

-- Auth provider enum: determines how the user authenticates
CREATE TYPE auth_provider AS ENUM ('email', 'google');

-- Conversation type enum
CREATE TYPE conversation_type AS ENUM ('private', 'group');

-- Member role enum
CREATE TYPE member_role AS ENUM ('admin', 'member');

-- Message type enum
CREATE TYPE message_type AS ENUM ('text', 'image', 'file', 'system');

-- Message status enum (delivery semantics)
CREATE TYPE message_status AS ENUM ('sent', 'delivered', 'read');

-- OTP purpose enum
CREATE TYPE otp_purpose AS ENUM ('email_verification', 'password_reset');


-- ==================== USERS ====================
-- Core user table supporting multi-provider authentication
-- auth_provider: 'email' = traditional email/password, 'google' = Google OAuth2
-- email_verified_at: NULL means unverified (OTP not yet confirmed)
-- google_id: populated only for Google OAuth users

CREATE TABLE users (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username        VARCHAR(50)  NOT NULL,
    email           VARCHAR(255) NOT NULL,
    password        VARCHAR(255),                                      -- NULL for Google OAuth users
    avatar          VARCHAR(500) DEFAULT '',
    auth_provider   auth_provider NOT NULL DEFAULT 'email',
    google_id       VARCHAR(255),                                      -- Google's unique user ID
    email_verified_at TIMESTAMPTZ,                                     -- NULL = not verified
    is_online       BOOLEAN DEFAULT FALSE,
    last_seen       TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ,                                       -- Soft delete

    -- Constraints
    CONSTRAINT uq_users_email    UNIQUE (email),
    CONSTRAINT uq_users_username UNIQUE (username),
    CONSTRAINT uq_users_google_id UNIQUE (google_id)
);

-- Indexes for users
CREATE INDEX idx_users_deleted_at ON users(deleted_at);
CREATE INDEX idx_users_auth_provider ON users(auth_provider);
CREATE INDEX idx_users_email_verified ON users(email_verified_at) WHERE email_verified_at IS NOT NULL;
CREATE INDEX idx_users_is_online ON users(is_online) WHERE is_online = TRUE;


-- ==================== OTP CODES ====================
-- One-Time Password codes for email verification and password reset
-- Each code expires after a configurable TTL (default 5 minutes)
-- used_at: marks the code as consumed, preventing replay attacks

CREATE TABLE otp_codes (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID         NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    code        VARCHAR(6)   NOT NULL,                                 -- 6-digit numeric code
    purpose     otp_purpose  NOT NULL DEFAULT 'email_verification',
    expires_at  TIMESTAMPTZ  NOT NULL,                                 -- When the code becomes invalid
    used_at     TIMESTAMPTZ,                                           -- NULL = not yet used
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- Index for fast lookup: find valid OTP for a user
CREATE INDEX idx_otp_user_purpose ON otp_codes(user_id, purpose, expires_at);
CREATE INDEX idx_otp_cleanup ON otp_codes(expires_at) WHERE used_at IS NULL;


-- ==================== CONVERSATIONS ====================
-- Supports both private (1-1) and group conversations
-- Private: name is empty, exactly 2 members
-- Group: has a name, creator, and unlimited members

CREATE TABLE conversations (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(100) DEFAULT '',                               -- Empty for private chats
    type        conversation_type NOT NULL DEFAULT 'private',
    avatar      VARCHAR(500) DEFAULT '',                               -- Group avatar
    creator_id  UUID REFERENCES users(id) ON DELETE SET NULL,          -- Group creator
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),                    -- Bumped on new message
    deleted_at  TIMESTAMPTZ                                            -- Soft delete
);

CREATE INDEX idx_conversations_deleted_at ON conversations(deleted_at);
CREATE INDEX idx_conversations_type ON conversations(type);
CREATE INDEX idx_conversations_updated_at ON conversations(updated_at DESC);


-- ==================== CONVERSATION MEMBERS ====================
-- Junction table linking users to conversations
-- Composite unique index prevents duplicate membership
-- Role determines admin privileges (group management)

CREATE TABLE conversation_members (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID        NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    user_id         UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role            member_role NOT NULL DEFAULT 'member',
    joined_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    muted_until     TIMESTAMPTZ,                                       -- Mute notifications until
    deleted_at      TIMESTAMPTZ,                                       -- Soft delete (left group)

    -- One user per conversation
    CONSTRAINT uq_conv_user UNIQUE (conversation_id, user_id)
);

CREATE INDEX idx_conv_members_user ON conversation_members(user_id);
CREATE INDEX idx_conv_members_conv ON conversation_members(conversation_id);
CREATE INDEX idx_conv_members_deleted_at ON conversation_members(deleted_at);


-- ==================== MESSAGES ====================
-- Core message table with support for:
-- - Text, image, file, and system messages
-- - Delivery status tracking (sent → delivered → read)
-- - File attachments (URL, name, size)
-- - Reply threading (reply_to_id)

CREATE TABLE messages (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID           NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    sender_id       UUID           NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    content         TEXT           DEFAULT '',
    type            message_type   NOT NULL DEFAULT 'text',
    status          message_status NOT NULL DEFAULT 'sent',
    file_url        VARCHAR(500)   DEFAULT '',
    file_name       VARCHAR(255)   DEFAULT '',
    file_size       BIGINT         DEFAULT 0,
    reply_to_id     UUID           REFERENCES messages(id) ON DELETE SET NULL,
    created_at      TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ                                        -- Soft delete
);

-- Indexes for messages (optimized for chat queries)
CREATE INDEX idx_messages_conversation ON messages(conversation_id, created_at DESC);
CREATE INDEX idx_messages_sender ON messages(sender_id);
CREATE INDEX idx_messages_deleted_at ON messages(deleted_at);
CREATE INDEX idx_messages_unread ON messages(conversation_id, sender_id, status)
    WHERE status != 'read' AND deleted_at IS NULL;


-- ==================== READ RECEIPTS ====================
-- Tracks exactly when each user reads each message
-- Used for "seen" indicators in the UI

CREATE TABLE read_receipts (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    message_id  UUID        NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    user_id     UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    read_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- One receipt per user per message
    CONSTRAINT uq_read_receipt UNIQUE (message_id, user_id)
);

CREATE INDEX idx_read_receipts_message ON read_receipts(message_id);
CREATE INDEX idx_read_receipts_user ON read_receipts(user_id);


-- ==================== TRIGGER: auto-update updated_at ====================

CREATE OR REPLACE FUNCTION trigger_set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER set_updated_at_users
    BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();

CREATE TRIGGER set_updated_at_conversations
    BEFORE UPDATE ON conversations
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();

CREATE TRIGGER set_updated_at_messages
    BEFORE UPDATE ON messages
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();
