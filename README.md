# Strength Tracker

A simple strength tracking application built with Go, Echo, htmx, Oat UI, and SQLite.

## Features

- Track individual exercise sets with weight (in kg), set number, and notes
- Normalized exercise names stored in a separate table
- Full CRUD operations (Create, Read, Update, Delete)
- Exercise history view showing all sets for a specific exercise
- htmx-powered inline delete with animations
- Clean, semantic UI with Oat CSS (zero dependency, ~8KB)
- No authentication (as requested)

## Architecture

```
internal/
├── db/           # SQLite wrapper + repository pattern
├── models/       # Data structures
├── handlers/     # HTTP handlers (Echo framework wrapped)
└── views/        # Templ templates

cmd/
└── server/       # Main entry point
```

### External Dependencies (Wrapped)

All external dependencies are wrapped in internal packages for easy replacement:

- **Echo** (`github.com/labstack/echo/v4`) - HTTP framework, wrapped in `internal/handlers`
- **SQLite** (`modernc.org/sqlite`) - Database, wrapped in `internal/db`
- **Templ** (`github.com/a-h/templ`) - Type-safe HTML templates
- **Oat CSS/JS** - UI styling via CDN
- **htmx** - Dynamic interactions via CDN

## Running the Application

### Prerequisites

- Go 1.21 or later

### Installation

```bash
# Clone or navigate to the project directory
cd stren

# Install templ CLI (for regenerating templates if needed)
go install github.com/a-h/templ/cmd/templ@latest

# Generate Go code from Templ templates (already done)
templ generate

# Build the application
go build -o stren ./cmd/server/main.go

# Run the application
./stren
```

The server will start on `http://localhost:8080`.

### Environment Variables

- `PORT` - Server port (default: 8080)
- `DB_PATH` - SQLite database file path (default: strength_tracker.db)

## Usage

1. **Dashboard** (`/`) - View recent workout history with all entries
2. **New Entry** (`/entries/new`) - Add a new exercise set:
   - Exercise name (autocomplete from existing exercises)
   - Set number
   - Weight in kg
   - Optional notes
3. **Exercise History** - Click on any exercise name to see all sets for that exercise
4. **Edit/Delete** - Each entry has edit and delete buttons (delete uses htmx for smooth removal)

## Data Model

### ExerciseType
Normalized exercise names to ensure consistency.

### ExerciseEntry
Individual set entries with:
- ID
- Exercise type reference
- Set number
- Weight (kg)
- Notes
- Timestamp

## Development

### Regenerating Templates

After modifying `.templ` files:

```bash
templ generate
```

### Database Schema

The database is automatically migrated on startup. Schema:

```sql
CREATE TABLE exercise_types (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT UNIQUE NOT NULL
);

CREATE TABLE exercise_entries (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    exercise_type_id INTEGER NOT NULL,
    set_number INTEGER NOT NULL,
    weight REAL NOT NULL,
    notes TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (exercise_type_id) REFERENCES exercise_types(id)
);
```

## License

MIT
