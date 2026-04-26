// Package controllers provides business logic for the strength tracker application.
// Controllers orchestrate between the HTTP routes layer and the data models,
// containing all domain logic that is independent of the web framework.
package controllers

import (
	"errors"

	"golang.org/x/crypto/bcrypt"

	"stren/internal/models"
	"stren/internal/utils"
)

// AuthController handles user authentication business logic.
type AuthController struct {
	userRepo   models.UserRepo
	jwtService *utils.JWTService
}

// NewAuthController creates a new AuthController instance.
func NewAuthController(userRepo models.UserRepo, jwtService *utils.JWTService) *AuthController {
	return &AuthController{
		userRepo:   userRepo,
		jwtService: jwtService,
	}
}

// Login validates credentials and returns the authenticated user and JWT token.
// Returns ErrInvalidCredentials if the email or password is incorrect.
func (ac *AuthController) Login(email, password string) (*models.User, string, error) {
	if email == "" || password == "" {
		return nil, "", ErrInvalidCredentials
	}

	user, err := ac.userRepo.GetUserByEmail(email)
	if err != nil {
		return nil, "", ErrInvalidCredentials
	}
	if user == nil {
		return nil, "", ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, "", ErrInvalidCredentials
	}

	token, err := ac.jwtService.GenerateToken(user.ID, user.Email, user.Name, user.IsAdmin)
	if err != nil {
		return nil, "", err
	}

	return user, token, nil
}

// Register creates a new user account and returns the user with a JWT token.
// Returns ErrEmailExists if the email is already registered.
func (ac *AuthController) Register(name, email, password string) (*models.User, string, error) {
	if name == "" || email == "" || password == "" {
		return nil, "", errors.New("all fields are required")
	}

	if len(password) < 6 {
		return nil, "", errors.New("password must be at least 6 characters")
	}

	existing, err := ac.userRepo.GetUserByEmail(email)
	if err != nil {
		return nil, "", errors.New("something went wrong")
	}
	if existing != nil {
		return nil, "", ErrEmailExists
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, "", errors.New("something went wrong")
	}

	user := &models.User{
		Name:         name,
		Email:        email,
		PasswordHash: string(hash),
	}
	if err := ac.userRepo.CreateUser(user); err != nil {
		return nil, "", errors.New("failed to create account")
	}

	token, err := ac.jwtService.GenerateToken(user.ID, user.Email, user.Name, user.IsAdmin)
	if err != nil {
		return nil, "", errors.New("failed to create session")
	}

	return user, token, nil
}
