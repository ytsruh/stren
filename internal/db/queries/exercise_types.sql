-- name: CreateType :one
INSERT INTO exercise_types (name)
VALUES (?)
ON CONFLICT(name) DO UPDATE SET name=name
RETURNING id;

-- name: GetTypeByName :one
SELECT id, name
FROM exercise_types
WHERE name = ?;

-- name: ListTypes :many
SELECT id, name
FROM exercise_types
ORDER BY name;

-- name: GetTypeByID :one
SELECT id, name
FROM exercise_types
WHERE id = ?;

-- name: UpdateType :one
UPDATE exercise_types
SET name = ?
WHERE id = ?
RETURNING id, name;
