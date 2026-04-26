-- +goose Up
-- Add users table and associate entries with users
CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    email TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    is_admin INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);

ALTER TABLE exercise_entries ADD COLUMN user_id INTEGER REFERENCES users(id);

CREATE INDEX IF NOT EXISTS idx_entries_user ON exercise_entries(user_id);

-- +goose Down
DROP INDEX IF EXISTS idx_entries_user;
ALTER TABLE exercise_entries DROP COLUMN user_id;
DROP INDEX IF EXISTS idx_users_email;
DROP TABLE IF EXISTS users;
