package models

import (
	"database/sql"
	"time"
)

// Repository defines the interface for exercise data access.
// This abstraction allows handlers to be tested with mock implementations
// without requiring a real database connection.
type Repository interface {
	// CreateType creates a new exercise type or returns the existing ID.
	// If tx is provided, the operation runs within the transaction.
	CreateType(tx *sql.Tx, name string) (int64, error)

	// GetTypeByName retrieves an exercise type by its normalized name.
	// Returns nil if not found.
	GetTypeByName(name string) (*ExerciseType, error)

	// ListTypes returns all exercise types ordered by name.
	ListTypes() ([]ExerciseType, error)

	// CreateEntry persists a new exercise entry and links it to its type.
	CreateEntry(entry *ExerciseEntry) error

	// GetEntry retrieves a single entry by ID with its exercise type name.
	// Returns nil if not found.
	GetEntry(id int64) (*ExerciseEntry, error)

	// UpdateEntry updates an existing entry without changing its created_at date.
	UpdateEntry(entry *ExerciseEntry) error

	// UpdateEntryWithDate updates an existing entry including its created_at date.
	UpdateEntryWithDate(entry *ExerciseEntry) error

	// DeleteEntry removes an entry by ID.
	DeleteEntry(id int64) error

	// ListEntries returns entries ordered by created_at descending.
	// If limit > 0, results are capped at that count.
	ListEntries(limit int) ([]ExerciseEntry, error)

	// GetEntriesByExercise returns all entries for a specific exercise name.
	GetEntriesByExercise(exerciseName string) ([]ExerciseEntry, error)

	// GetEntriesByDateRange returns entries within an inclusive date range.
	GetEntriesByDateRange(start, end time.Time) ([]ExerciseEntry, error)
}

// Compile-time check to ensure ExerciseRepository implements Repository.
var _ Repository = (*ExerciseRepository)(nil)
