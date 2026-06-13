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

// EntrySetInput describes a single set to be persisted as part of a multi-set
// entry submission. All sets within a submission share an exercise, user, notes
// and timestamp; only per-set values (reps, weight, rest) differ.
type EntrySetInput struct {
	Reps     int
	Weight   float64
	RestTime int
}

// CreateEntry creates a new exercise entry for the given user.
func (ec *EntryController) CreateEntry(userID string, exerciseID, notes string, createdAt time.Time, reps int, weight float64, restTime int) (*models.ExerciseEntry, error) {
	entry := &models.ExerciseEntry{
		ExerciseID: exerciseID,
		Notes:      notes,
		Reps:       reps,
		Weight:     weight,
		RestTime:   restTime,
		UserID:     userID,
		CreatedAt:  createdAt,
	}
	if err := ec.repo.CreateEntry(entry); err != nil {
		return nil, err
	}
	return entry, nil
}

// CreateEntries persists a group of sets in a single submission, all sharing
// the same exercise, user, notes and the supplied createdAt timestamp. The
// timestamp comes from the caller (typically parsed from the form's
// created_at field, or time.Now() when the field is empty) so a multi-set
// submission can be back-dated as a single unit. On the first repository
// error the loop aborts and the error is returned; partial-success semantics
// aren't worth the complexity for a workout log.
func (ec *EntryController) CreateEntries(userID, exerciseID, notes string, createdAt time.Time, sets []EntrySetInput) ([]models.ExerciseEntry, error) {
	created := make([]models.ExerciseEntry, 0, len(sets))
	for _, s := range sets {
		entry := &models.ExerciseEntry{
			ExerciseID: exerciseID,
			Notes:      notes,
			Reps:       s.Reps,
			Weight:     s.Weight,
			RestTime:   s.RestTime,
			UserID:     userID,
			CreatedAt:  createdAt,
		}
		if err := ec.repo.CreateEntry(entry); err != nil {
			return nil, err
		}
		created = append(created, *entry)
	}
	return created, nil
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

// ExerciseHistoryChartSize is the number of most-recent entries fetched to
// drive the line chart on the exercise history page. Kept relatively small
// because the chart sits in a narrow two-column grid cell on desktop, and
// only one point is plotted per calendar day (max weight) so beyond this
// size the additional points are usually redundant.
const ExerciseHistoryChartSize = 30

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

// GetRecentEntriesForChart returns the most recent ExerciseHistoryChartSize
// entries for the given exercise, scoped to the user. It feeds the line
// chart rendered on the exercise history page; the view groups the returned
// entries by day and plots the heaviest set of each day.
func (ec *EntryController) GetRecentEntriesForChart(exerciseID, userID string) ([]models.ExerciseEntry, error) {
	return ec.repo.GetEntriesByExercisePaginated(exerciseID, userID, ExerciseHistoryChartSize, 0)
}