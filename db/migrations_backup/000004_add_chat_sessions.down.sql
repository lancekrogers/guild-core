-- Drop chat sessions tables and indexes
DROP TRIGGER IF EXISTS update_session_timestamp;

DROP INDEX IF EXISTS idx_session_bookmarks_created;
DROP INDEX IF EXISTS idx_session_bookmarks_message;
DROP INDEX IF EXISTS idx_session_bookmarks_session;

DROP INDEX IF EXISTS idx_chat_messages_role;
DROP INDEX IF EXISTS idx_chat_messages_created;
DROP INDEX IF EXISTS idx_chat_messages_session;

DROP INDEX IF EXISTS idx_chat_sessions_updated;
DROP INDEX IF EXISTS idx_chat_sessions_campaign;

DROP TABLE IF EXISTS session_bookmarks;
DROP TABLE IF EXISTS chat_messages;
DROP TABLE IF EXISTS chat_sessions;