package controllers

import (
	"errors"
	"strings"

	"stren/internal/models"
)

// ErrNotFound is returned when an exercise is not found.
var ErrNotFound = errors.New("exercise not found")

// AdminController handles admin-only exercise operations.
type AdminController struct {
	repo models.AdminRepository
}

// NewAdminController creates a new AdminController instance.
func NewAdminController(repo models.AdminRepository) *AdminController {
	return &AdminController{repo: repo}
}

// List returns all exercises ordered by name.
func (ac *AdminController) List() ([]models.Exercise, error) {
	return ac.repo.List()
}

// Get retrieves a single exercise by ID.
func (ac *AdminController) Get(id int64) (*models.Exercise, error) {
	exercise, err := ac.repo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if exercise == nil {
		return nil, ErrNotFound
	}
	return exercise, nil
}

// Create creates a new exercise. Returns the created exercise.
func (ac *AdminController) Create(name string) (*models.Exercise, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("exercise name cannot be empty")
	}

	id, err := ac.repo.CreateNoTx(name)
	if err != nil {
		return nil, err
	}

	return &models.Exercise{ID: id, Name: name}, nil
}

// Update updates an existing exercise's name.
func (ac *AdminController) Update(id int64, name string) (*models.Exercise, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("exercise name cannot be empty")
	}

	exercise, err := ac.repo.Update(id, name)
	if err != nil {
		return nil, err
	}

	return exercise, nil
}