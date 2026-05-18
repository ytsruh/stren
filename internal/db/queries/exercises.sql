-- name: Create :one
INSERT INTO exercises (id, name)
VALUES (?, ?)
ON CONFLICT(name) DO UPDATE SET name=name
RETURNING id;

-- name: GetByName :one
SELECT id, name
FROM exercises
WHERE name = ?;

-- name: List :many
SELECT id, name
FROM exercises
ORDER BY name;

-- name: GetByID :one
SELECT id, name
FROM exercises
WHERE id = ?;

-- name: Update :one
UPDATE exercises
SET name = ?
WHERE id = ?
RETURNING id, name;