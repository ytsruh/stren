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
func (ac *AdminController) Get(id string) (*models.Exercise, error) {
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
func (ac *AdminController) Create(params models.CreateExerciseParams) (*models.Exercise, error) {
	params.Name = strings.TrimSpace(params.Name)
	if params.Name == "" {
		return nil, errors.New("exercise name cannot be empty")
	}

	if !params.Type.IsValid() {
		params.Type = models.ExerciseTypeOther
	}

	if !models.ValidateURL(params.VideoURL) {
		return nil, errors.New("video URL must be a valid URL")
	}

	if !models.ValidateURL(params.ImgURL) {
		return nil, errors.New("image URL must be a valid URL")
	}

	id, err := ac.repo.CreateNoTx(params)
	if err != nil {
		return nil, err
	}

	return &models.Exercise{
		ID:          id,
		Name:        params.Name,
		Description: params.Description,
		VideoURL:    params.VideoURL,
		ImgURL:      params.ImgURL,
		Type:        params.Type,
	}, nil
}

// Update updates an existing exercise's metadata.
func (ac *AdminController) Update(id string, params models.UpdateExerciseParams) (*models.Exercise, error) {
	params.Name = strings.TrimSpace(params.Name)
	if params.Name == "" {
		return nil, errors.New("exercise name cannot be empty")
	}

	if !params.Type.IsValid() {
		params.Type = models.ExerciseTypeOther
	}

	if !models.ValidateURL(params.VideoURL) {
		return nil, errors.New("video URL must be a valid URL")
	}

	if !models.ValidateURL(params.ImgURL) {
		return nil, errors.New("image URL must be a valid URL")
	}

	exercise, err := ac.repo.Update(id, params)
	if err != nil {
		return nil, err
	}

	return exercise, nil
}