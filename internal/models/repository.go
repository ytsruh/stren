package models

import (
	"database/sql"
	"time"
)

// Repository defines the interface for exercise data access.
// This abstraction allows handlers to be tested with mock implementations
// without requiring a real database connection.
type Repository interface {
	// Create creates a new exercise or returns the existing ID.
	// If tx is provided, the operation runs within the transaction.
	Create(tx *sql.Tx, name string) (string, error)

	// GetByName retrieves an exercise by its normalized name.
	// Returns nil if not found.
	GetByName(name string) (*Exercise, error)

	// GetExerciseByID retrieves an exercise by its UUID.
	// Returns nil if not found. Scopes to the given user ID.
	GetExerciseByID(id string, userID string) (*Exercise, error)

	// List returns all exercises ordered by name.
	List() ([]Exercise, error)

	// CreateEntry persists a new exercise entry and links it to its exercise.
	CreateEntry(entry *ExerciseEntry) error

	// GetEntry retrieves a single entry by ID with its exercise name.
	// Returns nil if not found. Scopes to the given user ID.
	GetEntry(id string, userID string) (*ExerciseEntry, error)

	// UpdateEntry updates an existing entry without changing its created_at date.
	// Scopes to the given user ID.
	UpdateEntry(entry *ExerciseEntry, userID string) error

	// UpdateEntryWithDate updates an existing entry including its created_at date.
	// Scopes to the given user ID.
	UpdateEntryWithDate(entry *ExerciseEntry, userID string) error

	// DeleteEntry removes an entry by ID. Scopes to the given user ID.
	DeleteEntry(id string, userID string) error

	// ListEntries returns entries ordered by created_at descending.
	// If limit > 0, results are capped at that count. Scopes to the given user ID.
	ListEntries(userID string, limit int) ([]ExerciseEntry, error)

	// GetEntriesByExercise returns all entries for a specific exercise ID.
	// Scopes to the given user ID.
	GetEntriesByExercise(exerciseID string, userID string) ([]ExerciseEntry, error)

	// GetEntriesByDateRange returns entries within an inclusive date range.
	// Scopes to the given user ID.
	GetEntriesByDateRange(start, end time.Time, userID string) ([]ExerciseEntry, error)

	// ListEntriesLast7Days returns entries from the last 7 days ordered by created_at descending.
	// Scopes to the given user ID.
	ListEntriesLast7Days(userID string) ([]ExerciseEntry, error)
}

// UserRepo defines the interface for user data access.
type UserRepo interface {
	CreateUser(user *User) error
	GetUserByEmail(email string) (*User, error)
	GetUserByID(id string) (*User, error)
	UpdateUser(user *User) error
}

// AdminUserRepo defines the interface for admin user operations.
type AdminUserRepo interface {
	ListUsers() ([]User, error)
}

// Compile-time check to ensure AdminUserRepository implements AdminUserRepo.
var _ AdminUserRepo = (*UserAdminRepository)(nil)

// Compile-time check to ensure ExerciseRepository implements Repository.
var _ Repository = (*ExerciseRepository)(nil)

// FeedbackRepoInterface defines the interface for feedback data access (used by controllers).
type FeedbackRepoInterface interface {
	Create(feedback *Feedback) error
	GetAll(filter string) ([]*Feedback, error)
	GetByID(id string) (*Feedback, error)
	UpdateStatus(id string, isClosed bool) error
}

// WeightRepo defines the interface for weight entry data access.
type WeightRepo interface {
	Create(entry *WeightEntry) error
	GetByID(id string, userID string) (*WeightEntry, error)
	List(userID string) ([]WeightEntry, error)
	Update(entry *WeightEntry, userID string) error
	Delete(id string, userID string) error
}

// Compile-time check to ensure WeightRepository implements WeightRepo.
var _ WeightRepo = (*WeightRepository)(nil)