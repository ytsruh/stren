CREATE TABLE exercises (
    id TEXT PRIMARY KEY,
    name TEXT UNIQUE NOT NULL,
    description TEXT,
    video_url TEXT,
    img_url TEXT,
    type TEXT NOT NULL DEFAULT 'other'
);

CREATE TABLE users (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    email TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    is_admin INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE exercise_entries (
    id TEXT PRIMARY KEY,
    exercise_id TEXT NOT NULL,
    user_id TEXT REFERENCES users(id),
    reps INTEGER NOT NULL,
    weight REAL NOT NULL,
    notes TEXT,
    rest_time INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (exercise_id) REFERENCES exercises(id)
);

CREATE INDEX idx_entries_exercise ON exercise_entries(exercise_id);
CREATE INDEX idx_entries_user ON exercise_entries(user_id);
CREATE INDEX idx_entries_created ON exercise_entries(created_at);
CREATE INDEX idx_users_email ON users(email);

CREATE TABLE feedback (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id),
    title TEXT NOT NULL,
    message TEXT NOT NULL,
    is_closed INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_feedback_user ON feedback(user_id);
CREATE INDEX idx_feedback_created ON feedback(created_at);
CREATE INDEX idx_feedback_closed ON feedback(is_closed);

CREATE TABLE weight_entries (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id),
    weight REAL NOT NULL,
    notes TEXT,
    photo_key TEXT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_weight_entries_user ON weight_entries(user_id);
CREATE INDEX idx_weight_entries_created ON weight_entries(created_at);