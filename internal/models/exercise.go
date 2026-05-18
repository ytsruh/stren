package models

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"stren/internal/db"
)

// ExerciseRepository provides CRUD operations for exercises using sqlc-generated queries.
type ExerciseRepository struct {
	db      *db.DB
	queries *db.Queries
}

// NewExerciseRepository creates a new exercise repository backed by sqlc.
func NewExerciseRepository(dbConn *db.DB) *ExerciseRepository {
	return &ExerciseRepository{
		db:      dbConn,
		queries: db.New(dbConn.Conn()),
	}
}

// Create creates a new exercise or returns the existing ID.
func (r *ExerciseRepository) Create(tx *sql.Tx, name string) (int64, error) {
	ctx := context.Background()
	if tx != nil {
		return r.queries.WithTx(tx).Create(ctx, name)
	}
	return r.queries.Create(ctx, name)
}

// GetByName retrieves an exercise by its normalized name.
func (r *ExerciseRepository) GetByName(name string) (*Exercise, error) {
	ctx := context.Background()
	row, err := r.queries.GetByName(ctx, name)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get exercise: %w", err)
	}
	return &Exercise{ID: row.ID, Name: row.Name}, nil
}

// List returns all exercises ordered by name.
func (r *ExerciseRepository) List() ([]Exercise, error) {
	ctx := context.Background()
	rows, err := r.queries.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list exercises: %w", err)
	}

	exercises := make([]Exercise, len(rows))
	for i, row := range rows {
		exercises[i] = Exercise{ID: row.ID, Name: row.Name}
	}
	return exercises, nil
}

// GetByID retrieves an exercise by its ID.
func (r *ExerciseRepository) GetByID(id int64) (*Exercise, error) {
	ctx := context.Background()
	row, err := r.queries.GetByID(ctx, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get exercise: %w", err)
	}
	return &Exercise{ID: row.ID, Name: row.Name}, nil
}

// CreateNoTx creates a new exercise without a transaction.
func (r *ExerciseRepository) CreateNoTx(name string) (int64, error) {
	ctx := context.Background()
	return r.queries.Create(ctx, name)
}

// Update updates an exercise's name.
func (r *ExerciseRepository) Update(id int64, name string) (*Exercise, error) {
	ctx := context.Background()
	row, err := r.queries.Update(ctx, db.UpdateParams{Name: name, ID: id})
	if err != nil {
		return nil, fmt.Errorf("failed to update exercise: %w", err)
	}
	return &Exercise{ID: row.ID, Name: row.Name}, nil
}

// CreateEntry persists a new exercise entry and links it to its exercise.
func (r *ExerciseRepository) CreateEntry(entry *ExerciseEntry) error {
	return r.db.Transaction(func(tx *sql.Tx) error {
		ctx := context.Background()
		qtx := r.queries.WithTx(tx)

		exerciseID, err := qtx.Create(ctx, entry.ExerciseName)
		if err != nil {
			return fmt.Errorf("failed to create exercise: %w", err)
		}

		id, err := qtx.CreateEntry(ctx, db.CreateEntryParams{
			ExerciseID: exerciseID,
			UserID:     sql.NullInt64{Int64: entry.UserID, Valid: true},
			Reps:       int64(entry.Reps),
			Weight:     entry.Weight,
			Notes:      stringToNullString(entry.Notes),
			CreatedAt:  timeToNullTime(entry.CreatedAt),
		})
		if err != nil {
			return fmt.Errorf("failed to create entry: %w", err)
		}

		entry.ID = id
		entry.ExerciseID = exerciseID
		return nil
	})
}

// GetEntry retrieves a single entry by ID with its exercise name.
// Returns nil if not found. Scopes to the given user ID.
func (r *ExerciseRepository) GetEntry(id int64, userID int64) (*ExerciseEntry, error) {
	ctx := context.Background()
	row, err := r.queries.GetEntry(ctx, db.GetEntryParams{
		ID:     id,
		UserID: sql.NullInt64{Int64: userID, Valid: true},
	})
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get entry: %w", err)
	}
	return mapGetEntryRow(row), nil
}

// UpdateEntry updates an existing entry without changing its created_at date.
// Scopes to the given user ID.
func (r *ExerciseRepository) UpdateEntry(entry *ExerciseEntry, userID int64) error {
	return r.db.Transaction(func(tx *sql.Tx) error {
		ctx := context.Background()
		qtx := r.queries.WithTx(tx)

		exerciseID, err := qtx.Create(ctx, entry.ExerciseName)
		if err != nil {
			return fmt.Errorf("failed to create exercise: %w", err)
		}

		err = qtx.UpdateEntry(ctx, db.UpdateEntryParams{
			ExerciseID: exerciseID,
			Reps:       int64(entry.Reps),
			Weight:     entry.Weight,
			Notes:      stringToNullString(entry.Notes),
			ID:         entry.ID,
			UserID:     sql.NullInt64{Int64: userID, Valid: true},
		})
		if err != nil {
			return fmt.Errorf("failed to update entry: %w", err)
		}

		entry.ExerciseID = exerciseID
		return nil
	})
}

// UpdateEntryWithDate updates an existing entry including its created_at date.
// Scopes to the given user ID.
func (r *ExerciseRepository) UpdateEntryWithDate(entry *ExerciseEntry, userID int64) error {
	return r.db.Transaction(func(tx *sql.Tx) error {
		ctx := context.Background()
		qtx := r.queries.WithTx(tx)

		exerciseID, err := qtx.Create(ctx, entry.ExerciseName)
		if err != nil {
			return fmt.Errorf("failed to create exercise: %w", err)
		}

		err = qtx.UpdateEntryWithDate(ctx, db.UpdateEntryWithDateParams{
			ExerciseID: exerciseID,
			Reps:       int64(entry.Reps),
			Weight:     entry.Weight,
			Notes:      stringToNullString(entry.Notes),
			CreatedAt:  timeToNullTime(entry.CreatedAt),
			ID:         entry.ID,
			UserID:     sql.NullInt64{Int64: userID, Valid: true},
		})
		if err != nil {
			return fmt.Errorf("failed to update entry: %w", err)
		}

		entry.ExerciseID = exerciseID
		return nil
	})
}

// DeleteEntry removes an entry by ID. Scopes to the given user ID.
func (r *ExerciseRepository) DeleteEntry(id int64, userID int64) error {
	ctx := context.Background()
	err := r.queries.DeleteEntry(ctx, db.DeleteEntryParams{
		ID:     id,
		UserID: sql.NullInt64{Int64: userID, Valid: true},
	})
	if err != nil {
		return fmt.Errorf("failed to delete entry: %w", err)
	}
	return nil
}

// ListEntries returns entries ordered by created_at descending.
// If limit > 0, results are capped at that count. Scopes to the given user ID.
func (r *ExerciseRepository) ListEntries(userID int64, limit int) ([]ExerciseEntry, error) {
	ctx := context.Background()

	var rows []db.ListEntriesRow
	var err error
	uid := sql.NullInt64{Int64: userID, Valid: true}
	if limit > 0 {
		var limited []db.ListEntriesWithLimitRow
		limited, err = r.queries.ListEntriesWithLimit(ctx, db.ListEntriesWithLimitParams{
			UserID: uid,
			Limit:  int64(limit),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to list entries: %w", err)
		}
		return mapListEntriesWithLimitRows(limited), nil
	}

	rows, err = r.queries.ListEntries(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("failed to list entries: %w", err)
	}
	return mapListEntriesRows(rows), nil
}

// GetEntriesByExercise returns all entries for a specific exercise.
// Scopes to the given user ID.
func (r *ExerciseRepository) GetEntriesByExercise(exerciseName string, userID int64) ([]ExerciseEntry, error) {
	ctx := context.Background()
	rows, err := r.queries.GetEntriesByExercise(ctx, db.GetEntriesByExerciseParams{
		Name:   exerciseName,
		UserID: sql.NullInt64{Int64: userID, Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get entries by exercise: %w", err)
	}
	return mapGetEntriesByExerciseRows(rows), nil
}

// GetEntriesByDateRange returns entries within an inclusive date range.
// Scopes to the given user ID.
func (r *ExerciseRepository) GetEntriesByDateRange(start, end time.Time, userID int64) ([]ExerciseEntry, error) {
	ctx := context.Background()
	rows, err := r.queries.GetEntriesByDateRange(ctx, db.GetEntriesByDateRangeParams{
		CreatedAt:   timeToNullTime(start),
		CreatedAt_2: timeToNullTime(end),
		UserID:      sql.NullInt64{Int64: userID, Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get entries by date range: %w", err)
	}
	return mapGetEntriesByDateRangeRows(rows), nil
}

// ListEntriesLast30Days returns entries from the last 30 days ordered by created_at descending.
// Scopes to the given user ID.
func (r *ExerciseRepository) ListEntriesLast30Days(userID int64) ([]ExerciseEntry, error) {
	ctx := context.Background()
	rows, err := r.queries.ListEntriesLast30Days(ctx, sql.NullInt64{Int64: userID, Valid: true})
	if err != nil {
		return nil, fmt.Errorf("failed to list entries last 30 days: %w", err)
	}
	return mapListEntriesLast30DaysRows(rows), nil
}

// --- Mapping helpers ---

func mapGetEntryRow(row db.GetEntryRow) *ExerciseEntry {
	return &ExerciseEntry{
		ID:           row.ID,
		ExerciseID:   row.ExerciseID,
		UserID:       nullInt64ToInt64(row.UserID),
		ExerciseName: row.ExerciseName,
		Reps:         int(row.Reps),
		Weight:       row.Weight,
		Notes:        nullStringToString(row.Notes),
		CreatedAt:    nullTimeToTime(row.CreatedAt),
	}
}

func mapListEntriesRows(rows []db.ListEntriesRow) []ExerciseEntry {
	entries := make([]ExerciseEntry, len(rows))
	for i, row := range rows {
		entries[i] = ExerciseEntry{
			ID:           row.ID,
			ExerciseID:   row.ExerciseID,
			UserID:       nullInt64ToInt64(row.UserID),
			ExerciseName: row.ExerciseName,
			Reps:         int(row.Reps),
			Weight:       row.Weight,
			Notes:        nullStringToString(row.Notes),
			CreatedAt:    nullTimeToTime(row.CreatedAt),
		}
	}
	return entries
}

func mapListEntriesWithLimitRows(rows []db.ListEntriesWithLimitRow) []ExerciseEntry {
	entries := make([]ExerciseEntry, len(rows))
	for i, row := range rows {
		entries[i] = ExerciseEntry{
			ID:           row.ID,
			ExerciseID:   row.ExerciseID,
			UserID:       nullInt64ToInt64(row.UserID),
			ExerciseName: row.ExerciseName,
			Reps:         int(row.Reps),
			Weight:       row.Weight,
			Notes:        nullStringToString(row.Notes),
			CreatedAt:    nullTimeToTime(row.CreatedAt),
		}
	}
	return entries
}

func mapGetEntriesByExerciseRows(rows []db.GetEntriesByExerciseRow) []ExerciseEntry {
	entries := make([]ExerciseEntry, len(rows))
	for i, row := range rows {
		entries[i] = ExerciseEntry{
			ID:           row.ID,
			ExerciseID:   row.ExerciseID,
			UserID:       nullInt64ToInt64(row.UserID),
			ExerciseName: row.ExerciseName,
			Reps:         int(row.Reps),
			Weight:       row.Weight,
			Notes:        nullStringToString(row.Notes),
			CreatedAt:    nullTimeToTime(row.CreatedAt),
		}
	}
	return entries
}

func mapGetEntriesByDateRangeRows(rows []db.GetEntriesByDateRangeRow) []ExerciseEntry {
	entries := make([]ExerciseEntry, len(rows))
	for i, row := range rows {
		entries[i] = ExerciseEntry{
			ID:           row.ID,
			ExerciseID:   row.ExerciseID,
			UserID:       nullInt64ToInt64(row.UserID),
			ExerciseName: row.ExerciseName,
			Reps:         int(row.Reps),
			Weight:       row.Weight,
			Notes:        nullStringToString(row.Notes),
			CreatedAt:    nullTimeToTime(row.CreatedAt),
		}
	}
	return entries
}

func mapListEntriesLast30DaysRows(rows []db.ListEntriesLast30DaysRow) []ExerciseEntry {
	entries := make([]ExerciseEntry, len(rows))
	for i, row := range rows {
		entries[i] = ExerciseEntry{
			ID:           row.ID,
			ExerciseID:   row.ExerciseID,
			UserID:       nullInt64ToInt64(row.UserID),
			ExerciseName: row.ExerciseName,
			Reps:         int(row.Reps),
			Weight:       row.Weight,
			Notes:        nullStringToString(row.Notes),
			CreatedAt:    nullTimeToTime(row.CreatedAt),
		}
	}
	return entries
}

func nullStringToString(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return ""
}

func stringToNullString(s string) sql.NullString {
	return sql.NullString{String: s, Valid: true}
}

func nullTimeToTime(nt sql.NullTime) time.Time {
	if nt.Valid {
		return nt.Time
	}
	return time.Time{}
}

func timeToNullTime(t time.Time) sql.NullTime {
	return sql.NullTime{Time: t, Valid: !t.IsZero()}
}

func nullInt64ToInt64(ni sql.NullInt64) int64 {
	if ni.Valid {
		return ni.Int64
	}
	return 0
}