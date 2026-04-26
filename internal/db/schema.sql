CREATE TABLE exercise_types (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT UNIQUE NOT NULL
);

CREATE TABLE exercise_entries (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    exercise_type_id INTEGER NOT NULL,
    reps INTEGER NOT NULL,
    weight REAL NOT NULL,
    notes TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (exercise_type_id) REFERENCES exercise_types(id)
);

CREATE INDEX idx_entries_type ON exercise_entries(exercise_type_id);
CREATE INDEX idx_entries_created ON exercise_entries(created_at);
