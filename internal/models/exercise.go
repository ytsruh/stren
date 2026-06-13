package models

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"stren/internal/db"

	"github.com/google/uuid"
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
func (r *ExerciseRepository) Create(tx *sql.Tx, name string) (string, error) {
	ctx := context.Background()
	id := uuid.New().String()
	if tx != nil {
		return r.queries.WithTx(tx).Create(ctx, db.CreateParams{ID: id, Name: name})
	}
	return r.queries.Create(ctx, db.CreateParams{ID: id, Name: name})
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
	return &Exercise{
		ID:          row.ID,
		Name:        row.Name,
		Description: nullStringToString(row.Description),
		VideoURL:    nullStringToString(row.VideoUrl),
		ImgURL:      nullStringToString(row.ImgUrl),
		Type:        ExerciseType(row.Type),
	}, nil
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
		exercises[i] = Exercise{
			ID:          row.ID,
			Name:        row.Name,
			Description: nullStringToString(row.Description),
			VideoURL:    nullStringToString(row.VideoUrl),
			ImgURL:      nullStringToString(row.ImgUrl),
			Type:        ExerciseType(row.Type),
		}
	}
	return exercises, nil
}

// GetByID retrieves an exercise by its ID.
func (r *ExerciseRepository) GetByID(id string) (*Exercise, error) {
	ctx := context.Background()
	row, err := r.queries.GetByID(ctx, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get exercise: %w", err)
	}
	return &Exercise{
		ID:          row.ID,
		Name:        row.Name,
		Description: nullStringToString(row.Description),
		VideoURL:    nullStringToString(row.VideoUrl),
		ImgURL:      nullStringToString(row.ImgUrl),
		Type:        ExerciseType(row.Type),
	}, nil
}

// GetExerciseByID retrieves an exercise by its ID, scoped to a user.
// Returns nil if not found.
func (r *ExerciseRepository) GetExerciseByID(id string, userID string) (*Exercise, error) {
	ctx := context.Background()
	row, err := r.queries.GetByID(ctx, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get exercise: %w", err)
	}
	return &Exercise{
		ID:          row.ID,
		Name:        row.Name,
		Description: nullStringToString(row.Description),
		VideoURL:    nullStringToString(row.VideoUrl),
		ImgURL:      nullStringToString(row.ImgUrl),
		Type:        ExerciseType(row.Type),
	}, nil
}

// CreateNoTx creates a new exercise without a transaction.
func (r *ExerciseRepository) CreateNoTx(params CreateExerciseParams) (string, error) {
	ctx := context.Background()
	id := uuid.New().String()
	return r.queries.Create(ctx, db.CreateParams{
		ID:          id,
		Name:        params.Name,
		Description: stringToNullString(params.Description),
		VideoUrl:    stringToNullString(params.VideoURL),
		ImgUrl:      stringToNullString(params.ImgURL),
		Type:        string(params.Type),
	})
}

// Update updates an exercise's metadata.
func (r *ExerciseRepository) Update(id string, params UpdateExerciseParams) (*Exercise, error) {
	ctx := context.Background()
	row, err := r.queries.Update(ctx, db.UpdateParams{
		Name:        params.Name,
		Description: stringToNullString(params.Description),
		VideoUrl:    stringToNullString(params.VideoURL),
		ImgUrl:      stringToNullString(params.ImgURL),
		Type:        string(params.Type),
		ID:          id,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update exercise: %w", err)
	}
	return &Exercise{
		ID:          row.ID,
		Name:        row.Name,
		Description: nullStringToString(row.Description),
		VideoURL:    nullStringToString(row.VideoUrl),
		ImgURL:      nullStringToString(row.ImgUrl),
		Type:        ExerciseType(row.Type),
	}, nil
}

// CreateEntry persists a new exercise entry and links it to its exercise.
func (r *ExerciseRepository) CreateEntry(entry *ExerciseEntry) error {
	return r.db.Transaction(func(tx *sql.Tx) error {
		ctx := context.Background()
		qtx := r.queries.WithTx(tx)

		entryUUID := uuid.New().String()
		_, err := qtx.CreateEntry(ctx, db.CreateEntryParams{
			ID:         entryUUID,
			ExerciseID: entry.ExerciseID,
			UserID:     sql.NullString{String: entry.UserID, Valid: true},
			Reps:       int64(entry.Reps),
			Weight:     entry.Weight,
			Notes:      stringToNullString(entry.Notes),
			RestTime:   int64(entry.RestTime),
			CreatedAt:  timeToNullTime(entry.CreatedAt),
		})
		if err != nil {
			return fmt.Errorf("failed to create entry: %w", err)
		}

		entry.ID = entryUUID
		return nil
	})
}

// GetEntry retrieves a single entry by ID with its exercise name.
// Returns nil if not found. Scopes to the given user ID.
func (r *ExerciseRepository) GetEntry(id string, userID string) (*ExerciseEntry, error) {
	ctx := context.Background()
	row, err := r.queries.GetEntry(ctx, db.GetEntryParams{
		ID:     id,
		UserID: sql.NullString{String: userID, Valid: true},
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
func (r *ExerciseRepository) UpdateEntry(entry *ExerciseEntry, userID string) error {
	return r.db.Transaction(func(tx *sql.Tx) error {
		ctx := context.Background()
		qtx := r.queries.WithTx(tx)

		err := qtx.UpdateEntry(ctx, db.UpdateEntryParams{
			ExerciseID: entry.ExerciseID,
			Reps:       int64(entry.Reps),
			Weight:     entry.Weight,
			Notes:      stringToNullString(entry.Notes),
			RestTime:   int64(entry.RestTime),
			ID:         entry.ID,
			UserID:     sql.NullString{String: userID, Valid: true},
		})
		if err != nil {
			return fmt.Errorf("failed to update entry: %w", err)
		}

		return nil
	})
}

// UpdateEntryWithDate updates an existing entry including its created_at date.
// Scopes to the given user ID.
func (r *ExerciseRepository) UpdateEntryWithDate(entry *ExerciseEntry, userID string) error {
	return r.db.Transaction(func(tx *sql.Tx) error {
		ctx := context.Background()
		qtx := r.queries.WithTx(tx)

		err := qtx.UpdateEntryWithDate(ctx, db.UpdateEntryWithDateParams{
			ExerciseID: entry.ExerciseID,
			Reps:       int64(entry.Reps),
			Weight:     entry.Weight,
			Notes:      stringToNullString(entry.Notes),
			RestTime:   int64(entry.RestTime),
			CreatedAt:  timeToNullTime(entry.CreatedAt),
			ID:         entry.ID,
			UserID:     sql.NullString{String: userID, Valid: true},
		})
		if err != nil {
			return fmt.Errorf("failed to update entry: %w", err)
		}

		return nil
	})
}

// DeleteEntry removes an entry by ID. Scopes to the given user ID.
func (r *ExerciseRepository) DeleteEntry(id string, userID string) error {
	ctx := context.Background()
	err := r.queries.DeleteEntry(ctx, db.DeleteEntryParams{
		ID:     id,
		UserID: sql.NullString{String: userID, Valid: true},
	})
	if err != nil {
		return fmt.Errorf("failed to delete entry: %w", err)
	}
	return nil
}

// ListEntries returns entries ordered by created_at descending.
// If limit > 0, results are capped at that count. Scopes to the given user ID.
func (r *ExerciseRepository) ListEntries(userID string, limit int) ([]ExerciseEntry, error) {
	ctx := context.Background()

	var rows []db.ListEntriesRow
	var err error
	uid := sql.NullString{String: userID, Valid: true}
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

// GetEntriesByExercisePaginated returns a page of entries for a specific exercise ID,
// ordered by created_at descending. Scopes to the given user ID.
func (r *ExerciseRepository) GetEntriesByExercisePaginated(exerciseID string, userID string, limit, offset int) ([]ExerciseEntry, error) {
	ctx := context.Background()
	rows, err := r.queries.GetEntriesByExercisePaginated(ctx, db.GetEntriesByExercisePaginatedParams{
		ExerciseID: exerciseID,
		UserID:     sql.NullString{String: userID, Valid: true},
		Limit:      int64(limit),
		Offset:     int64(offset),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get entries by exercise: %w", err)
	}
	return mapGetEntriesByExercisePaginatedRows(rows), nil
}

// GetMaxWeightByExercise returns the heaviest weight logged for the given exercise by
// the given user, or 0 when no entries exist. Scopes to the given user ID.
func (r *ExerciseRepository) GetMaxWeightByExercise(exerciseID string, userID string) (float64, error) {
	ctx := context.Background()
	max, err := r.queries.GetMaxWeightByExercise(ctx, db.GetMaxWeightByExerciseParams{
		ExerciseID: exerciseID,
		UserID:     sql.NullString{String: userID, Valid: true},
	})
	if err != nil {
		return 0, fmt.Errorf("failed to get max weight by exercise: %w", err)
	}
	return max, nil
}

// GetLastSetByExercise returns the most recent entry for the given exercise by the
// given user. Returns (nil, nil) when no entries exist. Scopes to the given user ID.
func (r *ExerciseRepository) GetLastSetByExercise(exerciseID string, userID string) (*ExerciseEntry, error) {
	ctx := context.Background()
	row, err := r.queries.GetLastSetByExercise(ctx, db.GetLastSetByExerciseParams{
		ExerciseID: exerciseID,
		UserID:     sql.NullString{String: userID, Valid: true},
	})
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get last set by exercise: %w", err)
	}
	return mapGetLastSetByExerciseRow(row), nil
}

// GetEntriesByDateRange returns entries within an inclusive date range.
// Scopes to the given user ID.
func (r *ExerciseRepository) GetEntriesByDateRange(start, end time.Time, userID string) ([]ExerciseEntry, error) {
	ctx := context.Background()
	rows, err := r.queries.GetEntriesByDateRange(ctx, db.GetEntriesByDateRangeParams{
		CreatedAt:   timeToNullTime(start),
		CreatedAt_2: timeToNullTime(end),
		UserID:      sql.NullString{String: userID, Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get entries by date range: %w", err)
	}
	return mapGetEntriesByDateRangeRows(rows), nil
}

// ListEntriesLast7Days returns entries from the last 7 days ordered by created_at descending.
// Scopes to the given user ID.
func (r *ExerciseRepository) ListEntriesLast7Days(userID string) ([]ExerciseEntry, error) {
	ctx := context.Background()
	rows, err := r.queries.ListEntriesLast7Days(ctx, sql.NullString{String: userID, Valid: true})
	if err != nil {
		return nil, fmt.Errorf("failed to list entries last 7 days: %w", err)
	}
	return mapListEntriesLast7DaysRows(rows), nil
}

// --- Mapping helpers ---

func mapGetEntryRow(row db.GetEntryRow) *ExerciseEntry {
	return &ExerciseEntry{
		ID:           row.ID,
		ExerciseID:   row.ExerciseID,
		UserID:       nullStringToString(row.UserID),
		ExerciseName: row.ExerciseName,
		Reps:         int(row.Reps),
		Weight:       row.Weight,
		Notes:        nullStringToString(row.Notes),
		RestTime:     int(row.RestTime),
		CreatedAt:    nullTimeToTime(row.CreatedAt),
	}
}

func mapListEntriesRows(rows []db.ListEntriesRow) []ExerciseEntry {
	entries := make([]ExerciseEntry, len(rows))
	for i, row := range rows {
		entries[i] = ExerciseEntry{
			ID:           row.ID,
			ExerciseID:   row.ExerciseID,
			UserID:       nullStringToString(row.UserID),
			ExerciseName: row.ExerciseName,
			Reps:         int(row.Reps),
			Weight:       row.Weight,
			Notes:        nullStringToString(row.Notes),
			RestTime:     int(row.RestTime),
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
			UserID:       nullStringToString(row.UserID),
			ExerciseName: row.ExerciseName,
			Reps:         int(row.Reps),
			Weight:       row.Weight,
			Notes:        nullStringToString(row.Notes),
			RestTime:     int(row.RestTime),
			CreatedAt:    nullTimeToTime(row.CreatedAt),
		}
	}
	return entries
}

func mapGetEntriesByExercisePaginatedRows(rows []db.GetEntriesByExercisePaginatedRow) []ExerciseEntry {
	entries := make([]ExerciseEntry, len(rows))
	for i, row := range rows {
		entries[i] = ExerciseEntry{
			ID:           row.ID,
			ExerciseID:   row.ExerciseID,
			UserID:       nullStringToString(row.UserID),
			ExerciseName: row.ExerciseName,
			Reps:         int(row.Reps),
			Weight:       row.Weight,
			Notes:        nullStringToString(row.Notes),
			RestTime:     int(row.RestTime),
			CreatedAt:    nullTimeToTime(row.CreatedAt),
		}
	}
	return entries
}

func mapGetLastSetByExerciseRow(row db.GetLastSetByExerciseRow) *ExerciseEntry {
	return &ExerciseEntry{
		ID:           row.ID,
		ExerciseID:   row.ExerciseID,
		UserID:       nullStringToString(row.UserID),
		ExerciseName: row.ExerciseName,
		Reps:         int(row.Reps),
		Weight:       row.Weight,
		Notes:        nullStringToString(row.Notes),
		RestTime:     int(row.RestTime),
		CreatedAt:    nullTimeToTime(row.CreatedAt),
	}
}

func mapGetEntriesByDateRangeRows(rows []db.GetEntriesByDateRangeRow) []ExerciseEntry {
	entries := make([]ExerciseEntry, len(rows))
	for i, row := range rows {
		entries[i] = ExerciseEntry{
			ID:           row.ID,
			ExerciseID:   row.ExerciseID,
			UserID:       nullStringToString(row.UserID),
			ExerciseName: row.ExerciseName,
			Reps:         int(row.Reps),
			Weight:       row.Weight,
			Notes:        nullStringToString(row.Notes),
			RestTime:     int(row.RestTime),
			CreatedAt:    nullTimeToTime(row.CreatedAt),
		}
	}
	return entries
}

func mapListEntriesLast7DaysRows(rows []db.ListEntriesLast7DaysRow) []ExerciseEntry {
	entries := make([]ExerciseEntry, len(rows))
	for i, row := range rows {
		entries[i] = ExerciseEntry{
			ID:           row.ID,
			ExerciseID:   row.ExerciseID,
			UserID:       nullStringToString(row.UserID),
			ExerciseName: row.ExerciseName,
			Reps:         int(row.Reps),
			Weight:       row.Weight,
			Notes:        nullStringToString(row.Notes),
			RestTime:     int(row.RestTime),
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
