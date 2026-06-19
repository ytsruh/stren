package controllers

import (
	"fmt"
	"log"
	"sort"
	"time"

	"stren/internal/models"
	"stren/internal/utils"
)

// WeightController handles body weight entry business logic.
type WeightController struct {
	repo models.WeightRepo
}

// NewWeightController creates a new WeightController instance.
func NewWeightController(repo models.WeightRepo) *WeightController {
	return &WeightController{repo: repo}
}

// ListWeightEntries returns all weight entries for a user ordered by date descending.
func (wc *WeightController) ListWeightEntries(userID string) ([]models.WeightEntry, error) {
	return wc.repo.List(userID)
}

// CreateWeightEntry creates a new weight entry for the given user with the current timestamp.
func (wc *WeightController) CreateWeightEntry(userID string, weight float64, notes string, photoKey string) (*models.WeightEntry, error) {
	entry := &models.WeightEntry{
		Weight:    weight,
		Notes:     notes,
		PhotoKey:  photoKey,
		UserID:    userID,
		CreatedAt: time.Now(),
	}
	if err := wc.repo.Create(entry); err != nil {
		return nil, err
	}
	return entry, nil
}

// GetWeightEntry retrieves a single weight entry by ID, scoped to the user.
func (wc *WeightController) GetWeightEntry(id, userID string) (*models.WeightEntry, error) {
	return wc.repo.GetByID(id, userID)
}

// GetWeightEntriesForCompare fetches two weight entries by ID for the
// image-comparison feature. Both must belong to the given user and both
// must have an associated photo. Returned slice is sorted by created_at
// ascending so callers can treat [0] as "before" and [1] as "after".
// Returns an error when the pair is incomplete, the entries are
// missing, or either entry lacks a photo — these are presented to the
// user as a toast, not as a 500.
func (wc *WeightController) GetWeightEntriesForCompare(idA, idB, userID string) ([]models.WeightEntry, error) {
	if idA == "" || idB == "" || idA == idB {
		return nil, fmt.Errorf("please choose two different weight entries to compare")
	}
	entries, err := wc.repo.GetByIDs(idA, idB, userID)
	if err != nil {
		return nil, err
	}
	if len(entries) < 2 {
		return nil, fmt.Errorf("could not find both weight entries")
	}
	for _, e := range entries {
		if !e.HasPhoto() {
			return nil, fmt.Errorf("both entries must have a photo to be compared")
		}
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].CreatedAt.Before(entries[j].CreatedAt)
	})
	return entries, nil
}

// UpdateWeightEntry updates an existing weight entry including its timestamp.
func (wc *WeightController) UpdateWeightEntry(id, userID string, weight float64, notes string, photoKey string, createdAt time.Time) (*models.WeightEntry, error) {
	entry := &models.WeightEntry{
		ID:        id,
		Weight:    weight,
		Notes:     notes,
		PhotoKey:  photoKey,
		UserID:    userID,
		CreatedAt: createdAt,
	}
	if err := wc.repo.Update(entry, userID); err != nil {
		return nil, err
	}
	return entry, nil
}

// DeleteWeightEntry removes a weight entry by ID, scoped to the user.
// If the entry had an associated photo in R2, it is deleted best-effort
// (errors are logged but do not block the DB delete).
func (wc *WeightController) DeleteWeightEntry(id, userID string) error {
	existing, err := wc.repo.GetByID(id, userID)
	if err != nil {
		return err
	}
	if existing != nil && existing.HasPhoto() {
		if delErr := utils.DeleteObject(existing.PhotoKey); delErr != nil {
			log.Printf("warning: failed to delete weight photo %q from R2: %v", existing.PhotoKey, delErr)
		}
	}
	return wc.repo.Delete(id, userID)
}
