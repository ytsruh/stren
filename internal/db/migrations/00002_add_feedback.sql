-- +goose Up
CREATE TABLE IF NOT EXISTS feedback (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL REFERENCES users(id),
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