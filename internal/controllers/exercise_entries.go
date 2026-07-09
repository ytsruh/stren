// Package controllers provides business logic for the strength tracker application.
package controllers

import (
	"math"
	"time"

	"stren/internal/models"
)

// ExerciseEntryController handles exercise entry business logic.
type ExerciseEntryController struct {
	repo models.Repository
}

// NewExerciseEntryController creates a new ExerciseEntryController instance.
func NewExerciseEntryController(repo models.Repository) *ExerciseEntryController {
	return &ExerciseEntryController{repo: repo}
}

// ListExerciseEntries returns the latest exercise entries for a user.
func (ec *ExerciseEntryController) ListExerciseEntries(userID string) ([]models.ExerciseEntry, error) {
	return ec.repo.ListExerciseEntries(userID, 100)
}

// ListExerciseEntriesLast7Days returns exercise entries from the last 7 days for a user.
func (ec *ExerciseEntryController) ListExerciseEntriesLast7Days(userID string) ([]models.ExerciseEntry, error) {
	return ec.repo.ListExerciseEntriesLast7Days(userID)
}

// GetExerciseEntry retrieves a single exercise entry by ID, scoped to the user.
func (ec *ExerciseEntryController) GetExerciseEntry(id, userID string) (*models.ExerciseEntry, error) {
	return ec.repo.GetExerciseEntry(id, userID)
}

// ExerciseSetInput describes a single set to be persisted as part of a multi-set
// exercise entry submission. All sets within a submission share an exercise,
// user, notes and timestamp; only per-set values (reps, weight, rest) differ.
type ExerciseSetInput struct {
	Reps     int
	Weight   float64
	RestTime int
}

// CreateExerciseEntry creates a new exercise entry for the given user.
func (ec *ExerciseEntryController) CreateExerciseEntry(userID string, exerciseID, notes string, createdAt time.Time, reps int, weight float64, restTime int) (*models.ExerciseEntry, error) {
	exerciseEntry := &models.ExerciseEntry{
		ExerciseID: exerciseID,
		Notes:      notes,
		Reps:       reps,
		Weight:     weight,
		RestTime:   restTime,
		UserID:     userID,
		CreatedAt:  createdAt,
	}
	if err := ec.repo.CreateExerciseEntry(exerciseEntry); err != nil {
		return nil, err
	}
	return exerciseEntry, nil
}

// CreateExerciseEntries persists a group of sets in a single submission, all
// sharing the same exercise, user, notes and the supplied createdAt timestamp.
// The timestamp comes from the caller (typically parsed from the form's
// created_at field, or time.Now() when the field is empty) so a multi-set
// submission can be back-dated as a single unit. On the first repository
// error the loop aborts and the error is returned; partial-success semantics
// aren't worth the complexity for a workout log.
func (ec *ExerciseEntryController) CreateExerciseEntries(userID, exerciseID, notes string, createdAt time.Time, sets []ExerciseSetInput) ([]models.ExerciseEntry, error) {
	created := make([]models.ExerciseEntry, 0, len(sets))
	for _, s := range sets {
		exerciseEntry := &models.ExerciseEntry{
			ExerciseID: exerciseID,
			Notes:      notes,
			Reps:       s.Reps,
			Weight:     s.Weight,
			RestTime:   s.RestTime,
			UserID:     userID,
			CreatedAt:  createdAt,
		}
		if err := ec.repo.CreateExerciseEntry(exerciseEntry); err != nil {
			return nil, err
		}
		created = append(created, *exerciseEntry)
	}
	return created, nil
}

// UpdateExerciseEntry updates an existing exercise entry, including its timestamp.
func (ec *ExerciseEntryController) UpdateExerciseEntry(id, userID string, exerciseID, notes string, reps int, weight float64, restTime int, createdAt time.Time) (*models.ExerciseEntry, error) {
	exerciseEntry := &models.ExerciseEntry{
		ID:         id,
		ExerciseID: exerciseID,
		Notes:      notes,
		Reps:       reps,
		Weight:     weight,
		RestTime:   restTime,
		UserID:     userID,
		CreatedAt:  createdAt,
	}
	if err := ec.repo.UpdateExerciseEntryWithDate(exerciseEntry, userID); err != nil {
		return nil, err
	}
	return exerciseEntry, nil
}

// DeleteExerciseEntry removes an exercise entry by ID, scoped to the user.
func (ec *ExerciseEntryController) DeleteExerciseEntry(id, userID string) error {
	return ec.repo.DeleteExerciseEntry(id, userID)
}

// List returns all exercises.
func (ec *ExerciseEntryController) List() ([]models.Exercise, error) {
	return ec.repo.List()
}

// ExerciseHistoryPageSize is the number of exercise entries shown per page on
// the exercise history view. Exposed as a constant so views and tests can rely on it.
const ExerciseHistoryPageSize = 25

// ExerciseHistoryChartSize is the number of most-recent exercise entries fetched
// to drive the line chart on the exercise history page. Kept relatively small
// because the chart sits in a narrow two-column grid cell on desktop, and
// only one point is plotted per calendar day (max weight) so beyond this
// size the additional points are usually redundant.
const ExerciseHistoryChartSize = 30

// GetExerciseEntriesByExercise returns a paginated page of exercise entries for
// a specific exercise ID along with the user's lifetime stats for that
// exercise. Pages are 1-indexed; invalid pages are clamped to 1. Scopes to
// the given user ID.
func (ec *ExerciseEntryController) GetExerciseEntriesByExercise(exerciseID string, userID string, page int) (*models.ExerciseHistoryPage, error) {
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * ExerciseHistoryPageSize

	// Fetch one extra row to detect whether a next page exists without a COUNT(*).
	rows, err := ec.repo.GetExerciseEntriesByExercisePaginated(exerciseID, userID, ExerciseHistoryPageSize+1, offset)
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
		ExerciseEntries: rows,
		Stats:           stats,
		Page:            page,
		HasPrev:         page > 1,
		HasNext:         hasNext,
	}, nil
}

// loadHistoryStats fetches the personal best and most recent set for the header
// stat cards. Always reflects the user's full history, not just the current page.
func (ec *ExerciseEntryController) loadHistoryStats(exerciseID string, userID string) (models.HistoryStats, error) {
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
func (ec *ExerciseEntryController) GetExerciseByID(id, userID string) (*models.Exercise, error) {
	return ec.repo.GetExerciseByID(id, userID)
}

// GetRecentExerciseEntriesForChart returns the most recent ExerciseHistoryChartSize
// exercise entries for the given exercise, scoped to the user. It feeds the
// line chart rendered on the exercise history page; the view groups the
// returned exercise entries by day and plots the heaviest set of each day.
func (ec *ExerciseEntryController) GetRecentExerciseEntriesForChart(exerciseID, userID string) ([]models.ExerciseEntry, error) {
	return ec.repo.GetExerciseEntriesByExercisePaginated(exerciseID, userID, ExerciseHistoryChartSize, 0)
}

// GetAllExerciseEntriesForChart returns every exercise entry the user has
// logged for the given exercise, scoped to the user. It feeds the dedicated
// /chart view, which renders a full-width line chart of the user's full
// workout history for that exercise. The view is responsible for aggregating
// to a daily series (heaviest weight per calendar day) before plotting.
// math.MaxInt32 is used as a sentinel for "no limit" — the underlying
// paginated repo method is reused to avoid introducing a new SQL query.
func (ec *ExerciseEntryController) GetAllExerciseEntriesForChart(exerciseID, userID string) ([]models.ExerciseEntry, error) {
	return ec.repo.GetExerciseEntriesByExercisePaginated(exerciseID, userID, math.MaxInt32, 0)
}
