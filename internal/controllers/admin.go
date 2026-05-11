package controllers

import (
	"errors"
	"strings"

	"stren/internal/models"
)

// ErrNotFound is returned when an exercise type is not found.
var ErrNotFound = errors.New("exercise type not found")

// AdminController handles admin-only exercise type operations.
type AdminController struct {
	repo models.AdminRepository
}

// NewAdminController creates a new AdminController instance.
func NewAdminController(repo models.AdminRepository) *AdminController {
	return &AdminController{repo: repo}
}

// ListTypes returns all exercise types ordered by name.
func (ac *AdminController) ListTypes() ([]models.ExerciseType, error) {
	return ac.repo.ListTypes()
}

// GetType retrieves a single exercise type by ID.
func (ac *AdminController) GetType(id int64) (*models.ExerciseType, error) {
	exerciseType, err := ac.repo.GetTypeByID(id)
	if err != nil {
		return nil, err
	}
	if exerciseType == nil {
		return nil, ErrNotFound
	}
	return exerciseType, nil
}

// CreateType creates a new exercise type. Returns the created type.
func (ac *AdminController) CreateType(name string) (*models.ExerciseType, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("exercise name cannot be empty")
	}

	id, err := ac.repo.CreateTypeNoTx(name)
	if err != nil {
		return nil, err
	}

	return &models.ExerciseType{ID: id, Name: name}, nil
}

// UpdateType updates an existing exercise type's name.
func (ac *AdminController) UpdateType(id int64, name string) (*models.ExerciseType, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("exercise name cannot be empty")
	}

	exerciseType, err := ac.repo.UpdateType(id, name)
	if err != nil {
		return nil, err
	}

	return exerciseType, nil
}