-- name: CreateSession :exec
INSERT INTO chat_sessions (id, name, campaign_id, metadata)
VALUES (?, ?, ?, ?);

-- name: GetSession :one
SELECT * FROM chat_sessions WHERE id = ?;

-- name: ListSessions :many
SELECT * FROM chat_sessions 
ORDER BY updated_at DESC
LIMIT ? OFFSET ?;

-- name: ListSessionsByCampaign :many
SELECT * FROM chat_sessions 
WHERE campaign_id = ?
ORDER BY updated_at DESC;

-- name: UpdateSession :exec
UPDATE chat_sessions
SET name = ?, metadata = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- name: DeleteSession :exec
DELETE FROM chat_sessions WHERE id = ?;

-- name: CountSessions :one
SELECT COUNT(*) FROM chat_sessions;

-- name: SearchSessions :many
SELECT * FROM chat_sessions
WHERE name LIKE ?
ORDER BY updated_at DESC
LIMIT ? OFFSET ?;

-- name: CreateMessage :exec
INSERT INTO chat_messages (id, session_id, role, content, tool_calls, metadata)
VALUES (?, ?, ?, ?, ?, ?);

-- name: GetMessage :one
SELECT * FROM chat_messages WHERE id = ?;

-- name: GetMessages :many
SELECT * FROM chat_messages
WHERE session_id = ?
ORDER BY created_at ASC;

-- name: GetMessagesPaginated :many
SELECT * FROM chat_messages
WHERE session_id = ?
ORDER BY created_at DESC
LIMIT ? OFFSET ?;

-- name: GetMessagesAfter :many
SELECT * FROM chat_messages
WHERE session_id = ? AND created_at > ?
ORDER BY created_at ASC;

-- name: CountMessages :one
SELECT COUNT(*) FROM chat_messages WHERE session_id = ?;

-- name: DeleteMessage :exec
DELETE FROM chat_messages WHERE id = ?;

-- name: SearchMessages :many
SELECT m.*, s.name as session_name, s.campaign_id
FROM chat_messages m
JOIN chat_sessions s ON m.session_id = s.id
WHERE m.content LIKE ?
ORDER BY m.created_at DESC
LIMIT ? OFFSET ?;

-- name: CreateBookmark :exec
INSERT INTO session_bookmarks (id, session_id, message_id, name)
VALUES (?, ?, ?, ?);

-- name: GetBookmark :one
SELECT * FROM session_bookmarks WHERE id = ?;

-- name: GetBookmarks :many
SELECT b.*, m.content as message_content, m.role
FROM session_bookmarks b
JOIN chat_messages m ON b.message_id = m.id
WHERE b.session_id = ?
ORDER BY b.created_at DESC;

-- name: DeleteBookmark :exec
DELETE FROM session_bookmarks WHERE id = ?;

-- name: GetBookmarksByMessage :many
SELECT * FROM session_bookmarks WHERE message_id = ?;