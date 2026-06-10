package controllers

import (
	"time"

	"stren/internal/models"
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
func (wc *WeightController) CreateWeightEntry(userID string, weight float64, notes string) (*models.WeightEntry, error) {
	entry := &models.WeightEntry{
		Weight:    weight,
		Notes:     notes,
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

// UpdateWeightEntry updates an existing weight entry including its timestamp.
func (wc *WeightController) UpdateWeightEntry(id, userID string, weight float64, notes string, createdAt time.Time) (*models.WeightEntry, error) {
	entry := &models.WeightEntry{
		ID:        id,
		Weight:    weight,
		Notes:     notes,
		UserID:    userID,
		CreatedAt: createdAt,
	}
	if err := wc.repo.Update(entry, userID); err != nil {
		return nil, err
	}
	return entry, nil
}

// DeleteWeightEntry removes a weight entry by ID, scoped to the user.
func (wc *WeightController) DeleteWeightEntry(id, userID string) error {
	return wc.repo.Delete(id, userID)
}