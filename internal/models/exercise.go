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

// CreateType creates a new exercise type or returns the existing ID.
func (r *ExerciseRepository) CreateType(tx *sql.Tx, name string) (int64, error) {
	ctx := context.Background()
	if tx != nil {
		return r.queries.WithTx(tx).CreateType(ctx, name)
	}
	return r.queries.CreateType(ctx, name)
}

// GetTypeByName retrieves an exercise type by its normalized name.
func (r *ExerciseRepository) GetTypeByName(name string) (*ExerciseType, error) {
	ctx := context.Background()
	row, err := r.queries.GetTypeByName(ctx, name)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get exercise type: %w", err)
	}
	return &ExerciseType{ID: row.ID, Name: row.Name}, nil
}

// ListTypes returns all exercise types ordered by name.
func (r *ExerciseRepository) ListTypes() ([]ExerciseType, error) {
	ctx := context.Background()
	rows, err := r.queries.ListTypes(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list exercise types: %w", err)
	}

	types := make([]ExerciseType, len(rows))
	for i, row := range rows {
		types[i] = ExerciseType{ID: row.ID, Name: row.Name}
	}
	return types, nil
}

// GetTypeByID retrieves an exercise type by its ID.
func (r *ExerciseRepository) GetTypeByID(id int64) (*ExerciseType, error) {
	ctx := context.Background()
	row, err := r.queries.GetTypeByID(ctx, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get exercise type: %w", err)
	}
	return &ExerciseType{ID: row.ID, Name: row.Name}, nil
}

// CreateTypeNoTx creates a new exercise type without a transaction.
func (r *ExerciseRepository) CreateTypeNoTx(name string) (int64, error) {
	ctx := context.Background()
	return r.queries.CreateType(ctx, name)
}

// UpdateType updates an exercise type's name.
func (r *ExerciseRepository) UpdateType(id int64, name string) (*ExerciseType, error) {
	ctx := context.Background()
	row, err := r.queries.UpdateType(ctx, db.UpdateTypeParams{Name: name, ID: id})
	if err != nil {
		return nil, fmt.Errorf("failed to update exercise type: %w", err)
	}
	return &ExerciseType{ID: row.ID, Name: row.Name}, nil
}

// CreateEntry persists a new exercise entry and links it to its type.
func (r *ExerciseRepository) CreateEntry(entry *ExerciseEntry) error {
	return r.db.Transaction(func(tx *sql.Tx) error {
		ctx := context.Background()
		qtx := r.queries.WithTx(tx)

		typeID, err := qtx.CreateType(ctx, entry.ExerciseName)
		if err != nil {
			return fmt.Errorf("failed to create exercise type: %w", err)
		}

		id, err := qtx.CreateEntry(ctx, db.CreateEntryParams{
			ExerciseTypeID: typeID,
			UserID:         sql.NullInt64{Int64: entry.UserID, Valid: true},
			Reps:           int64(entry.Reps),
			Weight:         entry.Weight,
			Notes:          stringToNullString(entry.Notes),
			CreatedAt:      timeToNullTime(entry.CreatedAt),
		})
		if err != nil {
			return fmt.Errorf("failed to create entry: %w", err)
		}

		entry.ID = id
		entry.ExerciseTypeID = typeID
		return nil
	})
}

// GetEntry retrieves a single entry by ID with its exercise type name.
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

		typeID, err := qtx.CreateType(ctx, entry.ExerciseName)
		if err != nil {
			return fmt.Errorf("failed to create exercise type: %w", err)
		}

		err = qtx.UpdateEntry(ctx, db.UpdateEntryParams{
			ExerciseTypeID: typeID,
			Reps:           int64(entry.Reps),
			Weight:         entry.Weight,
			Notes:          stringToNullString(entry.Notes),
			ID:             entry.ID,
			UserID:         sql.NullInt64{Int64: userID, Valid: true},
		})
		if err != nil {
			return fmt.Errorf("failed to update entry: %w", err)
		}

		entry.ExerciseTypeID = typeID
		return nil
	})
}

// UpdateEntryWithDate updates an existing entry including its created_at date.
// Scopes to the given user ID.
func (r *ExerciseRepository) UpdateEntryWithDate(entry *ExerciseEntry, userID int64) error {
	return r.db.Transaction(func(tx *sql.Tx) error {
		ctx := context.Background()
		qtx := r.queries.WithTx(tx)

		typeID, err := qtx.CreateType(ctx, entry.ExerciseName)
		if err != nil {
			return fmt.Errorf("failed to create exercise type: %w", err)
		}

		err = qtx.UpdateEntryWithDate(ctx, db.UpdateEntryWithDateParams{
			ExerciseTypeID: typeID,
			Reps:           int64(entry.Reps),
			Weight:         entry.Weight,
			Notes:          stringToNullString(entry.Notes),
			CreatedAt:      timeToNullTime(entry.CreatedAt),
			ID:             entry.ID,
			UserID:         sql.NullInt64{Int64: userID, Valid: true},
		})
		if err != nil {
			return fmt.Errorf("failed to update entry: %w", err)
		}

		entry.ExerciseTypeID = typeID
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

// --- Mapping helpers ---

func mapGetEntryRow(row db.GetEntryRow) *ExerciseEntry {
	return &ExerciseEntry{
		ID:             row.ID,
		ExerciseTypeID: row.ExerciseTypeID,
		UserID:         nullInt64ToInt64(row.UserID),
		ExerciseName:   row.ExerciseName,
		Reps:           int(row.Reps),
		Weight:         row.Weight,
		Notes:          nullStringToString(row.Notes),
		CreatedAt:      nullTimeToTime(row.CreatedAt),
	}
}

func mapListEntriesRows(rows []db.ListEntriesRow) []ExerciseEntry {
	entries := make([]ExerciseEntry, len(rows))
	for i, row := range rows {
		entries[i] = ExerciseEntry{
			ID:             row.ID,
			ExerciseTypeID: row.ExerciseTypeID,
			UserID:         nullInt64ToInt64(row.UserID),
			ExerciseName:   row.ExerciseName,
			Reps:           int(row.Reps),
			Weight:         row.Weight,
			Notes:          nullStringToString(row.Notes),
			CreatedAt:      nullTimeToTime(row.CreatedAt),
		}
	}
	return entries
}

func mapListEntriesWithLimitRows(rows []db.ListEntriesWithLimitRow) []ExerciseEntry {
	entries := make([]ExerciseEntry, len(rows))
	for i, row := range rows {
		entries[i] = ExerciseEntry{
			ID:             row.ID,
			ExerciseTypeID: row.ExerciseTypeID,
			UserID:         nullInt64ToInt64(row.UserID),
			ExerciseName:   row.ExerciseName,
			Reps:           int(row.Reps),
			Weight:         row.Weight,
			Notes:          nullStringToString(row.Notes),
			CreatedAt:      nullTimeToTime(row.CreatedAt),
		}
	}
	return entries
}

func mapGetEntriesByExerciseRows(rows []db.GetEntriesByExerciseRow) []ExerciseEntry {
	entries := make([]ExerciseEntry, len(rows))
	for i, row := range rows {
		entries[i] = ExerciseEntry{
			ID:             row.ID,
			ExerciseTypeID: row.ExerciseTypeID,
			UserID:         nullInt64ToInt64(row.UserID),
			ExerciseName:   row.ExerciseName,
			Reps:           int(row.Reps),
			Weight:         row.Weight,
			Notes:          nullStringToString(row.Notes),
			CreatedAt:      nullTimeToTime(row.CreatedAt),
		}
	}
	return entries
}

func mapGetEntriesByDateRangeRows(rows []db.GetEntriesByDateRangeRow) []ExerciseEntry {
	entries := make([]ExerciseEntry, len(rows))
	for i, row := range rows {
		entries[i] = ExerciseEntry{
			ID:             row.ID,
			ExerciseTypeID: row.ExerciseTypeID,
			UserID:         nullInt64ToInt64(row.UserID),
			ExerciseName:   row.ExerciseName,
			Reps:           int(row.Reps),
			Weight:         row.Weight,
			Notes:          nullStringToString(row.Notes),
			CreatedAt:      nullTimeToTime(row.CreatedAt),
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
