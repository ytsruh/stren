-- name: CreateFeedback :one
INSERT INTO feedback (user_id, title, message)
VALUES (?, ?, ?)
RETURNING id, user_id, title, message, is_closed, created_at, updated_at;

-- name: GetAll :many
SELECT f.id, f.user_id, f.title, f.message, f.is_closed, f.created_at, f.updated_at, u.name as user_name
FROM feedback f
LEFT JOIN users u ON f.user_id = u.id
ORDER BY f.created_at DESC;

-- name: GetAllOpen :many
SELECT f.id, f.user_id, f.title, f.message, f.is_closed, f.created_at, f.updated_at, u.name as user_name
FROM feedback f
LEFT JOIN users u ON f.user_id = u.id
WHERE f.is_closed = 0
ORDER BY f.created_at DESC;

-- name: GetAllClosed :many
SELECT f.id, f.user_id, f.title, f.message, f.is_closed, f.created_at, f.updated_at, u.name as user_name
FROM feedback f
LEFT JOIN users u ON f.user_id = u.id
WHERE f.is_closed = 1
ORDER BY f.created_at DESC;

-- name: GetFeedbackByID :one
SELECT f.id, f.user_id, f.title, f.message, f.is_closed, f.created_at, f.updated_at, u.name as user_name
FROM feedback f
LEFT JOIN users u ON f.user_id = u.id
WHERE f.id = ?;

-- name: UpdateStatus :one
UPDATE feedback
SET is_closed = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ?
RETURNING id, user_id, title, message, is_closed, created_at, updated_at;