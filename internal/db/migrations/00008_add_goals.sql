-- +goose Up
-- Goals are todo-style records with optional start, target, and end dates.
-- completed_at is nullable; when set, the goal is considered complete. The
-- start/target/end dates are all optional so the user can record any subset
-- (e.g. just a target date, or just a start date + completed_at).
CREATE TABLE goals (
    id           TEXT     PRIMARY KEY,
    user_id      TEXT     NOT NULL REFERENCES users(id),
    title        TEXT     NOT NULL,
    description  TEXT,
    start_date   DATETIME,
    target_date  DATETIME,
    end_date     DATETIME,
    completed_at DATETIME,
    created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_goals_user      ON goals(user_id);
-- The list page filters active vs completed in two separate queries
-- (ListActiveGoals and ListCompletedGoals); this index supports both.
CREATE INDEX idx_goals_completed ON goals(user_id, completed_at);

-- +goose Down
DROP INDEX IF EXISTS idx_goals_completed;
DROP INDEX IF EXISTS idx_goals_user;
DROP TABLE IF EXISTS goals;
