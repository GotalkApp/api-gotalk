-- ============================================
-- GoTalk: Rollback Initial Schema
-- Migration: 000001_init_schema (DOWN)
-- ============================================

-- Drop triggers
DROP TRIGGER IF EXISTS set_updated_at_messages ON messages;
DROP TRIGGER IF EXISTS set_updated_at_conversations ON conversations;
DROP TRIGGER IF EXISTS set_updated_at_users ON users;
DROP FUNCTION IF EXISTS trigger_set_updated_at();

-- Drop tables (order matters due to foreign keys)
DROP TABLE IF EXISTS read_receipts;
DROP TABLE IF EXISTS messages;
DROP TABLE IF EXISTS conversation_members;
DROP TABLE IF EXISTS conversations;
DROP TABLE IF EXISTS otp_codes;
DROP TABLE IF EXISTS users;

-- Drop enum types
DROP TYPE IF EXISTS otp_purpose;
DROP TYPE IF EXISTS message_status;
DROP TYPE IF EXISTS message_type;
DROP TYPE IF EXISTS member_role;
DROP TYPE IF EXISTS conversation_type;
DROP TYPE IF EXISTS auth_provider;
