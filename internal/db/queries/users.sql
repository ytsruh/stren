-- name: CreateUser :one
INSERT INTO users (id, name, email, password_hash, is_admin)
VALUES (?, ?, ?, ?, ?)
RETURNING id;

-- name: GetUserByEmail :one
SELECT id, name, email, password_hash, is_admin, target_weight, weight_unit, created_at, updated_at
FROM users
WHERE email = ?;

-- name: GetUserByID :one
SELECT id, name, email, password_hash, is_admin, target_weight, weight_unit, created_at, updated_at
FROM users
WHERE id = ?;

-- name: ListUsers :many
SELECT id, name, email, password_hash, is_admin, target_weight, weight_unit, created_at, updated_at
FROM users
ORDER BY created_at DESC;

-- name: UpdateUser :exec
UPDATE users
SET name = ?,
    target_weight = ?,
    weight_unit = ?,
    updated_at = ?
WHERE id = ?;
