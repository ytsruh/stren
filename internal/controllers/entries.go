// Package controllers provides business logic for the strength tracker application.
package controllers

import (
	"time"

	"stren/internal/models"
)

// EntryController handles exercise entry business logic.
type EntryController struct {
	repo models.Repository
}

// NewEntryController creates a new EntryController instance.
func NewEntryController(repo models.Repository) *EntryController {
	return &EntryController{repo: repo}
}

// ListEntries returns the latest entries for a user.
func (ec *EntryController) ListEntries(userID string) ([]models.ExerciseEntry, error) {
	return ec.repo.ListEntries(userID, 100)
}

// ListEntriesLast7Days returns entries from the last 7 days for a user.
func (ec *EntryController) ListEntriesLast7Days(userID string) ([]models.ExerciseEntry, error) {
	return ec.repo.ListEntriesLast7Days(userID)
}

// GetEntry retrieves a single entry by ID, scoped to the user.
func (ec *EntryController) GetEntry(id, userID string) (*models.ExerciseEntry, error) {
	return ec.repo.GetEntry(id, userID)
}

// CreateEntry creates a new exercise entry for the given user.
func (ec *EntryController) CreateEntry(userID string, exerciseID, notes string, reps int, weight float64, restTime int) (*models.ExerciseEntry, error) {
	entry := &models.ExerciseEntry{
		ExerciseID: exerciseID,
		Notes:      notes,
		Reps:       reps,
		Weight:     weight,
		RestTime:   restTime,
		UserID:     userID,
		CreatedAt:  time.Now(),
	}
	if err := ec.repo.CreateEntry(entry); err != nil {
		return nil, err
	}
	return entry, nil
}

// UpdateEntry updates an existing entry, including its timestamp.
func (ec *EntryController) UpdateEntry(id, userID string, exerciseID, notes string, reps int, weight float64, restTime int, createdAt time.Time) (*models.ExerciseEntry, error) {
	entry := &models.ExerciseEntry{
		ID:         id,
		ExerciseID: exerciseID,
		Notes:      notes,
		Reps:       reps,
		Weight:     weight,
		RestTime:   restTime,
		UserID:     userID,
		CreatedAt:  createdAt,
	}
	if err := ec.repo.UpdateEntryWithDate(entry, userID); err != nil {
		return nil, err
	}
	return entry, nil
}

// DeleteEntry removes an entry by ID, scoped to the user.
func (ec *EntryController) DeleteEntry(id, userID string) error {
	return ec.repo.DeleteEntry(id, userID)
}

// List returns all exercises.
func (ec *EntryController) List() ([]models.Exercise, error) {
	return ec.repo.List()
}

// ExerciseHistoryPageSize is the number of entries shown per page on the
// exercise history view. Exposed as a constant so views and tests can rely on it.
const ExerciseHistoryPageSize = 25

// GetEntriesByExercise returns a paginated page of entries for a specific exercise
// ID along with the user's lifetime stats for that exercise. Pages are 1-indexed;
// invalid pages are clamped to 1. Scopes to the given user ID.
func (ec *EntryController) GetEntriesByExercise(exerciseID string, userID string, page int) (*models.ExerciseHistoryPage, error) {
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * ExerciseHistoryPageSize

	// Fetch one extra row to detect whether a next page exists without a COUNT(*).
	rows, err := ec.repo.GetEntriesByExercisePaginated(exerciseID, userID, ExerciseHistoryPageSize+1, offset)
	if err != nil {
		return nil, err
	}
	hasNext := len(rows) > ExerciseHistoryPageSize
	if hasNext {
		rows = rows[:ExerciseHistoryPageSize]
	}

	stats, err := ec.loadHistoryStats(exerciseID, userID)
	if err != nil {
		return nil, err
	}

	return &models.ExerciseHistoryPage{
		Entries: rows,
		Stats:   stats,
		Page:    page,
		HasPrev: page > 1,
		HasNext: hasNext,
	}, nil
}

// loadHistoryStats fetches the personal best and most recent set for the header
// stat cards. Always reflects the user's full history, not just the current page.
func (ec *EntryController) loadHistoryStats(exerciseID string, userID string) (models.HistoryStats, error) {
	maxWeight, err := ec.repo.GetMaxWeightByExercise(exerciseID, userID)
	if err != nil {
		return models.HistoryStats{}, err
	}
	lastSet, err := ec.repo.GetLastSetByExercise(exerciseID, userID)
	if err != nil {
		return models.HistoryStats{}, err
	}
	stats := models.HistoryStats{MaxWeight: maxWeight}
	if lastSet != nil {
		stats.LastSet = *lastSet
	}
	return stats, nil
}

// GetExerciseByID returns an exercise by its UUID.
func (ec *EntryController) GetExerciseByID(id, userID string) (*models.Exercise, error) {
	return ec.repo.GetExerciseByID(id, userID)
}