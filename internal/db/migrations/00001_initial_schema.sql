-- +goose Up
-- Initial schema for strength tracker
CREATE TABLE IF NOT EXISTS exercises (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT UNIQUE NOT NULL
);

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

CREATE TABLE IF NOT EXISTS exercise_entries (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    exercise_id INTEGER NOT NULL,
    user_id INTEGER REFERENCES users(id),
    reps INTEGER NOT NULL,
    weight REAL NOT NULL,
    notes TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (exercise_id) REFERENCES exercises(id)
);

CREATE INDEX IF NOT EXISTS idx_entries_exercise ON exercise_entries(exercise_id);
CREATE INDEX IF NOT EXISTS idx_entries_user ON exercise_entries(user_id);
CREATE INDEX IF NOT EXISTS idx_entries_created ON exercise_entries(created_at);

-- +goose Down
DROP INDEX IF EXISTS idx_entries_created;
DROP INDEX IF EXISTS idx_entries_user;
DROP INDEX IF EXISTS idx_entries_exercise;
DROP TABLE IF EXISTS exercise_entries;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS exercises;