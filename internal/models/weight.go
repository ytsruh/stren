package models

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"stren/internal/db"

	"github.com/google/uuid"
)

// WeightEntry represents a single body weight entry for a user.
type WeightEntry struct {
	ID        string
	UserID    string
	Weight    float64
	Notes     string
	CreatedAt time.Time
}

// FormattedWeight returns the weight formatted as "X.X kg".
func (w *WeightEntry) FormattedWeight() string {
	return fmt.Sprintf("%.1f kg", w.Weight)
}

// FormattedDate returns the date in UK format (DD/MM/YY).
func (w *WeightEntry) FormattedDate() string {
	return w.CreatedAt.Format("02/01/06")
}

// WeightRepository provides CRUD operations for weight entries using sqlc-generated queries.
type WeightRepository struct {
	db      *db.DB
	queries *db.Queries
}

// NewWeightRepository creates a new weight repository backed by sqlc.
func NewWeightRepository(dbConn *db.DB) *WeightRepository {
	return &WeightRepository{
		db:      dbConn,
		queries: db.New(dbConn.Conn()),
	}
}

// Create persists a new weight entry.
func (r *WeightRepository) Create(entry *WeightEntry) error {
	ctx := context.Background()
	entryUUID := uuid.New().String()
	_, err := r.queries.CreateWeightEntry(ctx, db.CreateWeightEntryParams{
		ID:        entryUUID,
		UserID:    entry.UserID,
		Weight:    entry.Weight,
		Notes:     stringToNullString(entry.Notes),
		CreatedAt: entry.CreatedAt,
	})
	if err != nil {
		return fmt.Errorf("failed to create weight entry: %w", err)
	}
	entry.ID = entryUUID
	return nil
}

// GetByID retrieves a weight entry by ID scoped to a user.
// Returns nil if not found.
func (r *WeightRepository) GetByID(id string, userID string) (*WeightEntry, error) {
	ctx := context.Background()
	row, err := r.queries.GetWeightEntry(ctx, db.GetWeightEntryParams{
		ID:     id,
		UserID: userID,
	})
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get weight entry: %w", err)
	}
	return mapWeightEntryRow(row), nil
}

// List returns all weight entries for a user ordered by created_at descending.
func (r *WeightRepository) List(userID string) ([]WeightEntry, error) {
	ctx := context.Background()
	rows, err := r.queries.ListWeightEntries(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list weight entries: %w", err)
	}
	return mapWeightEntryRows(rows), nil
}

// Update updates an existing weight entry including its created_at date.
// Scopes to the given user ID.
func (r *WeightRepository) Update(entry *WeightEntry, userID string) error {
	ctx := context.Background()
	err := r.queries.UpdateWeightEntry(ctx, db.UpdateWeightEntryParams{
		Weight:    entry.Weight,
		Notes:     stringToNullString(entry.Notes),
		CreatedAt: entry.CreatedAt,
		ID:        entry.ID,
		UserID:    userID,
	})
	if err != nil {
		return fmt.Errorf("failed to update weight entry: %w", err)
	}
	return nil
}

// Delete removes a weight entry by ID. Scopes to the given user ID.
func (r *WeightRepository) Delete(id string, userID string) error {
	ctx := context.Background()
	err := r.queries.DeleteWeightEntry(ctx, db.DeleteWeightEntryParams{
		ID:     id,
		UserID: userID,
	})
	if err != nil {
		return fmt.Errorf("failed to delete weight entry: %w", err)
	}
	return nil
}

// --- Mapping helpers ---

func mapWeightEntryRow(row db.WeightEntry) *WeightEntry {
	return &WeightEntry{
		ID:        row.ID,
		UserID:    row.UserID,
		Weight:    row.Weight,
		Notes:     nullStringToString(row.Notes),
		CreatedAt: row.CreatedAt,
	}
}

func mapWeightEntryRows(rows []db.WeightEntry) []WeightEntry {
	entries := make([]WeightEntry, len(rows))
	for i, row := range rows {
		entries[i] = WeightEntry{
			ID:        row.ID,
			UserID:    row.UserID,
			Weight:    row.Weight,
			Notes:     nullStringToString(row.Notes),
			CreatedAt: row.CreatedAt,
		}
	}
	return entries
}