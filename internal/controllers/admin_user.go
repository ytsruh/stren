package controllers

import (
	"stren/internal/models"
)

// AdminUserController handles admin-only user operations.
type AdminUserController struct {
	repo models.AdminUserRepo
}

// NewAdminUserController creates a new AdminUserController instance.
func NewAdminUserController(repo models.AdminUserRepo) *AdminUserController {
	return &AdminUserController{repo: repo}
}

// ListUsers returns all users ordered by creation date (newest first).
func (uc *AdminUserController) ListUsers() ([]models.User, error) {
	return uc.repo.ListUsers()
}
