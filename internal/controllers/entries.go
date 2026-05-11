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
func (ec *EntryController) ListEntries(userID int64) ([]models.ExerciseEntry, error) {
	return ec.repo.ListEntries(userID, 100)
}

// GetEntry retrieves a single entry by ID, scoped to the user.
func (ec *EntryController) GetEntry(id, userID int64) (*models.ExerciseEntry, error) {
	return ec.repo.GetEntry(id, userID)
}

// CreateEntry creates a new exercise entry for the given user.
func (ec *EntryController) CreateEntry(userID int64, exerciseName, notes string, reps int, weight float64) (*models.ExerciseEntry, error) {
	entry := &models.ExerciseEntry{
		ExerciseName: exerciseName,
		Notes:        notes,
		Reps:         reps,
		Weight:       weight,
		UserID:       userID,
		CreatedAt:    time.Now(),
	}
	if err := ec.repo.CreateEntry(entry); err != nil {
		return nil, err
	}
	return entry, nil
}

// UpdateEntry updates an existing entry, including its timestamp.
func (ec *EntryController) UpdateEntry(id, userID int64, exerciseName, notes string, reps int, weight float64, createdAt time.Time) (*models.ExerciseEntry, error) {
	entry := &models.ExerciseEntry{
		ID:           id,
		ExerciseName: exerciseName,
		Notes:        notes,
		Reps:         reps,
		Weight:       weight,
		UserID:       userID,
		CreatedAt:    createdAt,
	}
	if err := ec.repo.UpdateEntryWithDate(entry, userID); err != nil {
		return nil, err
	}
	return entry, nil
}

// DeleteEntry removes an entry by ID, scoped to the user.
func (ec *EntryController) DeleteEntry(id, userID int64) error {
	return ec.repo.DeleteEntry(id, userID)
}

// List returns all exercises.
func (ec *EntryController) List() ([]models.Exercise, error) {
	return ec.repo.List()
}

// GetEntriesByExercise returns all entries for a specific exercise name.
func (ec *EntryController) GetEntriesByExercise(exerciseName string, userID int64) ([]models.ExerciseEntry, error) {
	return ec.repo.GetEntriesByExercise(exerciseName, userID)
}