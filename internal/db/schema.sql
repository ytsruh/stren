CREATE TABLE exercise_types (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT UNIQUE NOT NULL
);

CREATE TABLE users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    email TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    is_admin INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE exercise_entries (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    exercise_type_id INTEGER NOT NULL,
    user_id INTEGER REFERENCES users(id),
    reps INTEGER NOT NULL,
    weight REAL NOT NULL,
    notes TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (exercise_type_id) REFERENCES exercise_types(id)
);

CREATE INDEX idx_entries_type ON exercise_entries(exercise_type_id);
CREATE INDEX idx_entries_user ON exercise_entries(user_id);
CREATE INDEX idx_entries_created ON exercise_entries(created_at);
CREATE INDEX idx_users_email ON users(email);
