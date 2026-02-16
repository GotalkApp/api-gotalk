CREATE TABLE IF NOT EXISTS message_attachments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    message_id UUID NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    type VARCHAR(20) NOT NULL,
    url VARCHAR(1000) NOT NULL,
    file_name VARCHAR(255),
    file_size BIGINT DEFAULT 0,
    mime_type VARCHAR(100),
    width INT DEFAULT 0,
    height INT DEFAULT 0,
    duration DOUBLE PRECISION DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_message_attachments_message_id ON message_attachments(message_id);
