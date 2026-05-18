-- +goose Up
-- Combined initial schema for strength tracker
CREATE TABLE IF NOT EXISTS exercises (
    id TEXT PRIMARY KEY,
    name TEXT UNIQUE NOT NULL
);

CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    email TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    is_admin INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);

CREATE TABLE IF NOT EXISTS exercise_entries (
    id TEXT PRIMARY KEY,
    exercise_id TEXT NOT NULL,
    user_id TEXT REFERENCES users(id),
    reps INTEGER NOT NULL,
    weight REAL NOT NULL,
    notes TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (exercise_id) REFERENCES exercises(id)
);

CREATE INDEX IF NOT EXISTS idx_entries_exercise ON exercise_entries(exercise_id);
CREATE INDEX IF NOT EXISTS idx_entries_user ON exercise_entries(user_id);
CREATE INDEX IF NOT EXISTS idx_entries_created ON exercise_entries(created_at);

CREATE TABLE IF NOT EXISTS feedback (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id),
    title TEXT NOT NULL,
    message TEXT NOT NULL,
    is_closed INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_feedback_user ON feedback(user_id);
CREATE INDEX IF NOT EXISTS idx_feedback_created ON feedback(created_at);
CREATE INDEX IF NOT EXISTS idx_feedback_closed ON feedback(is_closed);

-- +goose Down
DROP INDEX IF EXISTS idx_feedback_closed;
DROP INDEX IF EXISTS idx_feedback_created;
DROP INDEX IF NOT EXISTS idx_feedback_user;
DROP TABLE IF EXISTS feedback;
DROP INDEX IF EXISTS idx_entries_created;
DROP INDEX IF EXISTS idx_entries_user;
DROP INDEX IF EXISTS idx_entries_exercise;
DROP TABLE IF EXISTS exercise_entries;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS exercises;